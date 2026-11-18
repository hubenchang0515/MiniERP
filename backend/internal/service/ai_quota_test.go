package service

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"minierp/internal/database"
	"minierp/internal/model"
)

func TestAIQuotaWeekAccounting(t *testing.T) {
	s := newService(t)
	ctx := t.Context()
	user := model.User{Username: "member", Enabled: true}
	if err := s.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	// One second before Monday in Beijing, in a week that crosses New Year.
	sunday := time.Date(2026, 12, 27, 15, 59, 59, 0, time.UTC)
	monday := sunday.Add(time.Second)
	nextWeek := monday.AddDate(0, 0, 7)
	balance := func(at time.Time) AIQuotaBalance {
		t.Helper()
		result, err := s.AIQuotaBalance(ctx, user.ID, at)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	key := model.AIKey{ID: 1, UserID: user.ID}
	mapping := model.AIModel{InternalID: "chat", Pricing: model.AIModelPricing{
		InputPerMillion: 1_000, OutputPerMillion: 4_000, ReservePoints: 7,
	}}
	if initial := balance(sunday); initial.Enabled || initial.Configured || initial.Week != "2026-12-21" || !initial.ResetsAt.Equal(monday) {
		t.Fatalf("initial balance: %+v", initial)
	}
	if pending, err := s.ReserveAIQuota(ctx, "unmetered", key, mapping, sunday); err != nil || pending != nil {
		t.Fatalf("default should not charge: %v %v", pending, err)
	}
	check(s.SetAIQuotaEnabled(ctx, true))
	_, err := s.ReserveAIQuota(ctx, "no-profile", key, mapping, sunday)
	wantStatus(t, err, 403)
	check(s.SetAIWeeklyQuota(ctx, user.ID, 7))
	check(s.GrantAIQuota(ctx, user.ID, 1, 1, "本周项目", sunday))

	var wg sync.WaitGroup
	results := make(chan *model.AIQuotaCharge, 12)
	for i := range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			anotherKey := model.AIKey{ID: uint(i + 1), UserID: user.ID}
			charge, err := s.ReserveAIQuota(ctx, fmt.Sprintf("parallel-%d", i), anotherKey, mapping, sunday)
			if err != nil {
				wantStatus(t, err, 403)
			}
			results <- charge
		}()
	}
	wg.Wait()
	close(results)
	var charge *model.AIQuotaCharge
	for reservation := range results {
		if reservation == nil {
			continue
		}
		if charge != nil {
			t.Fatal("the same funds were reserved twice")
		}
		charge = reservation
	}
	if charge == nil {
		t.Fatal("no request reserved the available balance")
	}
	if got := balance(sunday); got.Available != AIPointMicros || got.Reserved != 7*AIPointMicros || got.Used != 0 {
		t.Fatalf("first week balance: %+v", got)
	}
	check(s.MarkAIQuotaDispatched(ctx, charge.RequestID))
	check(s.SettleAIQuota(ctx, charge.RequestID, &AITokenUsage{Input: 5_000, Output: 1_000}))
	check(s.SettleAIQuota(ctx, charge.RequestID, nil))
	if got := balance(sunday); got.Available != -AIPointMicros || got.Used != 9*AIPointMicros || got.Reserved != 0 {
		t.Fatalf("cross-week settlement and overrun: %+v", got)
	}
	if got := balance(monday); got.Week != "2026-12-28" || got.Available != 7*AIPointMicros || got.Added != 0 ||
		!got.ResetsAt.Equal(nextWeek) || got.Used != 0 ||
		got.TotalUsage != (AIQuotaUsageTotal{InputTokens: 5_000, OutputTokens: 1_000, Charged: 9 * AIPointMicros}) {
		t.Fatalf("new week: %+v", got)
	}
	if got := balance(nextWeek); got.Week != "2027-01-04" || got.Available != 7*AIPointMicros {
		t.Fatalf("new year week: %+v", got)
	}
	check(s.SetAIWeeklyQuota(ctx, user.ID, 10))
	if got := balance(monday); got.Available != 10*AIPointMicros {
		t.Fatalf("updated allowance must apply immediately: %+v", got)
	}
	var grant model.AIQuotaGrant
	check(s.DB.First(&grant).Error)
	if grant.Reason != "本周项目" || grant.Week != "2026-12-21" || grant.AdminID != 1 {
		t.Fatalf("grant audit missing: %+v", grant)
	}
}

func TestAIQuotaRecoveryAfterReopen(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	db, err := database.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	closeDB := func() {
		pool, _ := db.DB()
		_ = pool.Close()
	}
	defer closeDB()
	s := &Service{DB: db}
	now := time.Now()
	user := model.User{Username: "member", Enabled: true}
	for _, err := range []error{db.Create(&user).Error, s.SetAIWeeklyQuota(ctx, user.ID, 10), s.SetAIQuotaEnabled(ctx, true)} {
		if err != nil {
			t.Fatal(err)
		}
	}
	mapping := model.AIModel{InternalID: "chat", Pricing: model.AIModelPricing{ReservePoints: 4}}
	key := model.AIKey{ID: 1, UserID: user.ID}
	for _, id := range []string{"before-send", "sent"} {
		if _, err := s.ReserveAIQuota(ctx, id, key, mapping, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.MarkAIQuotaDispatched(ctx, "sent"); err != nil {
		t.Fatal(err)
	}
	closeDB()
	db, err = database.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s = &Service{DB: db}
	for range 2 {
		if err := s.RecoverAIQuota(ctx); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.AIQuotaBalance(ctx, user.ID, now)
	if err != nil || got.Reserved != 0 || got.Used != 4*AIPointMicros || got.Available != 6*AIPointMicros {
		t.Fatalf("recovered balance: %+v, %v", got, err)
	}
	var charges []model.AIQuotaCharge
	if err := db.Order("request_id").Find(&charges).Error; err != nil || len(charges) != 2 ||
		charges[0].Status != quotaReleased || charges[1].Status != quotaEstimated {
		t.Fatalf("recovered ledger: %+v, %v", charges, err)
	}
}
