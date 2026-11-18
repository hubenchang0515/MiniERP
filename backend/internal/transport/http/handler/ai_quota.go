package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"minierp/internal/model"
	"minierp/internal/service"
)

func (h *Handler) MyAIQuota(c *gin.Context) {
	h.respondAIQuota(c, c.MustGet("user").(model.User).ID)
}

func (h *Handler) AIQuotaUsers(c *gin.Context) {
	page, size, ok := pagination(c)
	if !ok {
		return
	}
	items, total, err := h.Service.ListAIQuotaUsers(c.Request.Context(), c.Query("q"), page, size, time.Now())
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "pageSize": size})
}

func (h *Handler) UserAIQuota(c *gin.Context) {
	if userID, ok := id(c); ok {
		h.respondAIQuota(c, userID)
	}
}

func (h *Handler) respondAIQuota(c *gin.Context, userID uint) {
	balance, err := h.Service.AIQuotaBalance(c.Request.Context(), userID, time.Now())
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, balance)
}

func (h *Handler) AIQuotaSettings(c *gin.Context) {
	enabled, err := h.Service.AIQuotaEnabled(c.Request.Context())
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, gin.H{"enabled": enabled})
}

func (h *Handler) SetAIQuotaEnabled(c *gin.Context) {
	var in struct {
		Enabled *bool `json:"enabled"`
	}
	if !bind(c, &in) {
		return
	}
	if in.Enabled == nil {
		RespondError(c, service.Fail(400, "validation", "请指定是否启用 AI 额度"))
		return
	}
	if err := h.Service.SetAIQuotaEnabled(c.Request.Context(), *in.Enabled); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}

func (h *Handler) SetAIWeeklyQuota(c *gin.Context) {
	userID, ok := id(c)
	if !ok {
		return
	}
	var in struct {
		WeeklyPoints *int64 `json:"weeklyPoints"`
	}
	if !bind(c, &in) {
		return
	}
	if in.WeeklyPoints == nil {
		RespondError(c, service.Fail(400, "validation", "请填写周额度"))
		return
	}
	if err := h.Service.SetAIWeeklyQuota(c.Request.Context(), userID, *in.WeeklyPoints); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}

func (h *Handler) GrantAIQuota(c *gin.Context) {
	userID, ok := id(c)
	if !ok {
		return
	}
	var in struct {
		Points int64  `json:"points"`
		Reason string `json:"reason"`
	}
	if !bind(c, &in) {
		return
	}
	adminID := c.MustGet("user").(model.User).ID
	if err := h.Service.GrantAIQuota(c.Request.Context(), userID, adminID, in.Points, in.Reason, time.Time{}); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}

func (h *Handler) MyAIUsage(c *gin.Context) {
	page, size, ok := pagination(c)
	if !ok {
		return
	}
	userID := c.MustGet("user").(model.User).ID
	db := h.Service.ReadDB().WithContext(c.Request.Context()).
		Model(&model.AIQuotaCharge{}).Where("user_id = ?", userID)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		RespondError(c, err)
		return
	}
	items := []model.AIQuotaCharge{}
	if err := db.Order("created_at DESC, request_id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "pageSize": size})
}
