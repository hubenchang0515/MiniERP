package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"minierp/internal/model"
)

const (
	AIPointMicros  int64 = 1_000_000
	MaxAIPoints    int64 = 1_000_000_000
	quotaSettingID       = 1

	quotaReserved   = "reserved"
	quotaDispatched = "dispatched"
	quotaActual     = "actual"
	quotaEstimated  = "estimated"
	quotaReleased   = "released"
)

// Quota periods use Beijing's current UTC+08:00 offset on every deployment.
var quotaZone = time.FixedZone("Beijing", 8*60*60)

func aiQuotaWeek(now time.Time) (string, time.Time) {
	local := now.In(quotaZone)
	daysSinceMonday := (int(local.Weekday()) + 6) % 7
	start := time.Date(local.Year(), local.Month(), local.Day()-daysSinceMonday, 0, 0, 0, 0, quotaZone)
	return start.Format(time.DateOnly), start.AddDate(0, 0, 7)
}

type AIQuotaBalance struct {
	Enabled      bool              `json:"enabled"`
	Configured   bool              `json:"configured"`
	Week         string            `json:"week"`
	ResetsAt     time.Time         `json:"resetsAt"`
	WeeklyPoints int64             `json:"weeklyPoints"`
	Added        int64             `json:"addedMicros,string"`
	Used         int64             `json:"usedMicros,string"`
	Reserved     int64             `json:"reservedMicros,string"`
	Available    int64             `json:"availableMicros,string"`
	TotalUsage   AIQuotaUsageTotal `json:"totalUsage" gorm:"-"`
}

type AIQuotaUsageTotal struct {
	InputTokens  int64 `json:"inputTokens,string"`
	OutputTokens int64 `json:"outputTokens,string"`
	Charged      int64 `json:"chargedMicros,string"`
}

type AIQuotaUser struct {
	AIQuotaBalance
	UserID      uint   `json:"userId"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type AITokenUsage struct {
	Input, Output int64
}

func aiQuotaEnabled(db *gorm.DB) (bool, error) {
	var setting model.AIQuotaSetting
	err := db.Find(&setting, quotaSettingID).Error
	return setting.Enabled, err
}

func (s *Service) AIQuotaEnabled(ctx context.Context) (bool, error) {
	return aiQuotaEnabled(s.ReadDB().WithContext(ctx))
}

func (s *Service) SetAIQuotaEnabled(ctx context.Context, enabled bool) error {
	return s.DB.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&model.AIQuotaSetting{ID: quotaSettingID, Enabled: enabled}).Error
}

func (s *Service) AIQuotaBalance(ctx context.Context, userID uint, now time.Time) (AIQuotaBalance, error) {
	week, resetsAt := aiQuotaWeek(now)
	result := AIQuotaBalance{Week: week, ResetsAt: resetsAt}
	users := s.ReadDB().WithContext(ctx).Table("users AS u")
	err := withAIQuotaWeek(users, week).Select(aiQuotaBalanceColumns).
		Where("u.id = ?", userID).Take(&result).Error
	if err != nil {
		return result, Normalize(err)
	}
	result.Available = quotaAvailable(result.WeeklyPoints, result.Added, result.Used, result.Reserved)
	totals, err := s.aiQuotaUsageTotals(ctx, []uint{userID})
	result.TotalUsage = totals[userID]
	return result, err
}

const aiQuotaBalanceColumns = "COALESCE(setting.enabled, 0) AS enabled, q.user_id IS NOT NULL AS configured, " +
	"COALESCE(q.weekly_points, 0) AS weekly_points, COALESCE(w.added, 0) AS added, " +
	"COALESCE(w.used, 0) AS used, COALESCE(w.reserved, 0) AS reserved"

func withAIQuotaWeek(users *gorm.DB, week string) *gorm.DB {
	return users.
		Joins("LEFT JOIN ai_quota_settings AS setting ON setting.id = ?", quotaSettingID).
		Joins("LEFT JOIN ai_user_quota AS q ON q.user_id = u.id").
		Joins("LEFT JOIN ai_quota_weeks AS w ON w.user_id = u.id AND w.week = ?", week)
}

func (s *Service) ListAIQuotaUsers(ctx context.Context, search string, page, size int, now time.Time) ([]AIQuotaUser, int64, error) {
	week, resetsAt := aiQuotaWeek(now)
	users := s.ReadDB().WithContext(ctx).Table("users AS u")
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		users = users.Where("(u.username LIKE ? OR u.display_name LIKE ?)", like, like)
	}
	var total int64
	if err := users.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := []AIQuotaUser{}
	if err := withAIQuotaWeek(users, week).Select("u.id AS user_id, u.username, u.display_name, " + aiQuotaBalanceColumns).
		Order("u.id").Offset((page - 1) * size).Limit(size).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	userIDs := make([]uint, len(items))
	for i := range items {
		userIDs[i] = items[i].UserID
	}
	totals, err := s.aiQuotaUsageTotals(ctx, userIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		item := &items[i]
		item.Week, item.ResetsAt = week, resetsAt
		item.Available = quotaAvailable(item.WeeklyPoints, item.Added, item.Used, item.Reserved)
		item.TotalUsage = totals[item.UserID]
	}
	return items, total, nil
}

// Aggregate only users on the requested page, using the existing ledger index.
func (s *Service) aiQuotaUsageTotals(ctx context.Context, userIDs []uint) (map[uint]AIQuotaUsageTotal, error) {
	totals := make(map[uint]AIQuotaUsageTotal, len(userIDs))
	if len(userIDs) == 0 {
		return totals, nil
	}
	var rows []struct {
		UserID uint
		AIQuotaUsageTotal
	}
	err := s.ReadDB().WithContext(ctx).Model(&model.AIQuotaCharge{}).
		Select("user_id, COALESCE(SUM(input_tokens), 0) AS input_tokens, "+
			"COALESCE(SUM(output_tokens), 0) AS output_tokens, COALESCE(SUM(charged), 0) AS charged").
		Where("user_id IN ? AND status IN ?", userIDs, []string{quotaActual, quotaEstimated}).
		Group("user_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		totals[row.UserID] = row.AIQuotaUsageTotal
	}
	return totals, nil
}

func quotaAvailable(weeklyPoints, added, used, reserved int64) int64 {
	return weeklyPoints*AIPointMicros + added - used - reserved
}

func (s *Service) SetAIWeeklyQuota(ctx context.Context, userID uint, points int64) error {
	if points < 0 || points > MaxAIPoints {
		return Fail(400, "validation", "周额度必须是 0–10 亿之间的整数积分")
	}
	return Normalize(s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Select("id").First(&user, userID).Error; err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{UpdateAll: true}).
			Create(&model.AIUserQuota{UserID: userID, WeeklyPoints: points}).Error
	}))
}

func (s *Service) GrantAIQuota(ctx context.Context, userID, adminID uint, points int64, reason string, now time.Time) error {
	reason = strings.TrimSpace(reason)
	if points <= 0 || points > MaxAIPoints || reason == "" || utf8.RuneCountInString(reason) > 256 {
		return Fail(400, "validation", "补发须填写 1–10 亿之间的整数积分及原因（最多 256 字）")
	}
	return Normalize(s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if now.IsZero() {
			now = time.Now()
		}
		week, _ := aiQuotaWeek(now)
		var settings model.AIUserQuota
		if err := tx.First(&settings, "user_id = ?", userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return Fail(400, "quota_unconfigured", "请先给该用户设置周额度再补发")
			}
			return err
		}
		bucket, err := loadQuotaWeek(tx, userID, week)
		if err != nil {
			return err
		}
		added := points * AIPointMicros
		if bucket.Added > MaxAIPoints*AIPointMicros-added {
			return Fail(400, "quota_grant_limit", "本周累计补发不能超过 10 亿积分")
		}
		if err := quotaWeekQuery(tx, userID, week).UpdateColumn("added", bucket.Added+added).Error; err != nil {
			return err
		}
		grant := model.AIQuotaGrant{UserID: userID, AdminID: adminID, Week: week, Points: points, Reason: reason, CreatedAt: now}
		return tx.Create(&grant).Error
	}))
}

func quotaWeekQuery(tx *gorm.DB, userID uint, week string) *gorm.DB {
	return tx.Model(&model.AIQuotaWeek{}).Where("user_id = ? AND week = ?", userID, week)
}

func loadQuotaWeek(tx *gorm.DB, userID uint, week string) (model.AIQuotaWeek, error) {
	bucket := model.AIQuotaWeek{UserID: userID, Week: week}
	err := quotaWeekQuery(tx, userID, week).FirstOrCreate(&bucket).Error
	return bucket, err
}

// ReserveAIQuota returns nil when enforcement is off. The read-only fast path
// keeps ordinary proxy calls independent of the SQLite writer.
func (s *Service) ReserveAIQuota(ctx context.Context, requestID string, key model.AIKey, mapping model.AIModel, now time.Time) (*model.AIQuotaCharge, error) {
	enabled, err := s.AIQuotaEnabled(ctx)
	if err != nil || !enabled {
		return nil, err
	}

	var charge *model.AIQuotaCharge
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		enabled, err := aiQuotaEnabled(tx)
		if err != nil || !enabled {
			return err
		}
		price := mapping.Pricing
		if price.ReservePoints <= 0 {
			return Fail(403, "ai_model_unpriced", "此模型尚未设置积分价格，请联系管理员")
		}
		var settings model.AIUserQuota
		if err := tx.Find(&settings, "user_id = ?", key.UserID).Error; err != nil {
			return err
		}
		if settings.UserID == 0 {
			return Fail(403, "insufficient_quota", "尚未分配 AI 周额度，请联系管理员")
		}
		if now.IsZero() {
			now = time.Now()
		}
		week, _ := aiQuotaWeek(now)
		bucket, err := loadQuotaWeek(tx, key.UserID, week)
		if err != nil {
			return err
		}
		reserved := price.ReservePoints * AIPointMicros
		if quotaAvailable(settings.WeeklyPoints, bucket.Added, bucket.Used, bucket.Reserved) < reserved {
			return Fail(403, "insufficient_quota", "本周可用积分不足以预留此请求，请等待已有请求完成或联系管理员")
		}
		if err := quotaWeekQuery(tx, key.UserID, week).UpdateColumn("reserved", bucket.Reserved+reserved).Error; err != nil {
			return err
		}
		charge = &model.AIQuotaCharge{
			RequestID: requestID, UserID: key.UserID, KeyID: key.ID, Week: week, Model: mapping.InternalID,
			Reserved: reserved, Status: quotaReserved, InputPerMillion: price.InputPerMillion,
			OutputPerMillion: price.OutputPerMillion, CreatedAt: now,
		}
		return tx.Create(charge).Error
	})
	if err != nil {
		return nil, err
	}
	return charge, nil
}

func (s *Service) MarkAIQuotaDispatched(ctx context.Context, requestID string) error {
	return s.changeQuotaDispatch(ctx, requestID, quotaReserved, quotaDispatched)
}

func (s *Service) MarkAIQuotaRejected(ctx context.Context, requestID string) error {
	return s.changeQuotaDispatch(ctx, requestID, quotaDispatched, quotaReserved)
}

func (s *Service) changeQuotaDispatch(ctx context.Context, requestID, from, to string) error {
	result := s.DB.WithContext(ctx).Model(&model.AIQuotaCharge{}).
		Where("request_id = ? AND status = ?", requestID, from).UpdateColumn("status", to)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("unexpected AI quota dispatch state")
	}
	return nil
}

func (s *Service) SettleAIQuota(ctx context.Context, requestID string, usage *AITokenUsage) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var charge model.AIQuotaCharge
		if err := tx.First(&charge, "request_id = ?", requestID).Error; err != nil {
			return err
		}
		if charge.Status != quotaReserved && charge.Status != quotaDispatched {
			return nil
		}

		charged, status := int64(0), quotaReleased
		update := map[string]any{}
		if charge.Status == quotaDispatched {
			charged, status = charge.Reserved, quotaEstimated
			if cost, ok := aiUsageCost(charge, usage); ok {
				charged, status = cost, quotaActual
				update["input_tokens"], update["output_tokens"] = usage.Input, usage.Output
			}
		}
		// Guard SQLite integer arithmetic, including later subtraction of reserves.
		maxUsed := int64(math.MaxInt64) - 2*MaxAIPoints*AIPointMicros
		result := quotaWeekQuery(tx, charge.UserID, charge.Week).
			Where("reserved >= ? AND used <= ?", charge.Reserved, maxUsed-charged).
			UpdateColumns(map[string]any{
				"reserved": gorm.Expr("reserved - ?", charge.Reserved),
				"used":     gorm.Expr("used + ?", charged),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("AI quota balance invariant violated")
		}
		update["status"], update["charged"] = status, charged
		return tx.Model(&charge).UpdateColumns(update).Error
	})
}

func aiUsageCost(charge model.AIQuotaCharge, usage *AITokenUsage) (int64, bool) {
	if usage == nil || usage.Input < 0 || usage.Output < 0 {
		return 0, false
	}
	// One integral point per million tokens equals one micropoint per token.
	if charge.InputPerMillion > 0 && usage.Input > math.MaxInt64/charge.InputPerMillion {
		return 0, false
	}
	input := usage.Input * charge.InputPerMillion
	if charge.OutputPerMillion > 0 && usage.Output > (math.MaxInt64-input)/charge.OutputPerMillion {
		return 0, false
	}
	cost := input + usage.Output*charge.OutputPerMillion
	return cost, cost <= math.MaxInt64-2*MaxAIPoints*AIPointMicros
}

// Run before serving requests so only records from a previous process are recovered.
func (s *Service) RecoverAIQuota(ctx context.Context) error {
	for {
		var requestIDs []string
		err := s.DB.WithContext(ctx).Model(&model.AIQuotaCharge{}).
			Where("status IN ?", []string{quotaReserved, quotaDispatched}).Limit(100).
			Pluck("request_id", &requestIDs).Error
		if err != nil || len(requestIDs) == 0 {
			return err
		}
		for _, requestID := range requestIDs {
			if err := s.SettleAIQuota(ctx, requestID, nil); err != nil {
				return err
			}
		}
	}
}
