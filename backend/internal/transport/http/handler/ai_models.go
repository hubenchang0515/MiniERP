package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"minierp/internal/model"
	"minierp/internal/service"
)

func availableAIModels(s *service.Service) *gorm.DB {
	return s.ReadDB().Model(&model.AIModel{}).Joins("JOIN ai_resources ON ai_resources.id = ai_models.resource_id").Where("ai_models.enabled = ? AND ai_resources.enabled = ?", true, true)
}

func (h *Handler) AIModels(c *gin.Context) {
	page, size, ok := pagination(c)
	if !ok {
		return
	}
	q := h.Service.ReadDB().WithContext(c.Request.Context()).Model(&model.AIModel{}).Joins("JOIN ai_resources ON ai_resources.id = ai_models.resource_id")
	if text := c.Query("q"); text != "" {
		q = q.Where("ai_models.internal_id LIKE ? OR ai_models.external_id LIKE ? OR ai_resources.name LIKE ?", "%"+text+"%", "%"+text+"%", "%"+text+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		RespondError(c, err)
		return
	}
	items := []struct {
		model.AIModel
		ResourceName    string `json:"resourceName"`
		ResourceEnabled bool   `json:"resourceEnabled"`
	}{}
	if err := q.Select("ai_models.*, ai_resources.name AS resource_name, ai_resources.enabled AS resource_enabled").Order("ai_models.id DESC").Offset((page - 1) * size).Limit(size).Scan(&items).Error; err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "pageSize": size})
}

func (h *Handler) SaveAIModel(c *gin.Context) {
	var modelID uint
	if c.Param("id") != "" {
		var ok bool
		modelID, ok = id(c)
		if !ok {
			return
		}
	}
	var in service.AIModelInput
	if !bind(c, &in) {
		return
	}
	m, err := h.Service.SaveAIModel(modelID, in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, m)
}

func (h *Handler) DeleteAIModel(c *gin.Context) {
	modelID, ok := id(c)
	if !ok {
		return
	}
	result := h.Service.DB.Delete(&model.AIModel{}, modelID)
	if result.Error != nil {
		RespondError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		RespondError(c, service.Fail(404, "not_found", "模型映射不存在"))
		return
	}
	c.Status(204)
}
