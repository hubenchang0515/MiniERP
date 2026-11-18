package handler

import (
	"github.com/gin-gonic/gin"
	"minioa/internal/model"
	"minioa/internal/service"
)

func (h *Handler) AIProviderKeys(c *gin.Context) {
	page, size, ok := pagination(c)
	if !ok {
		return
	}
	q := h.Service.ReadDB().WithContext(c.Request.Context()).Model(&model.AIProviderKey{}).Table(h.Service.DB.NamingStrategy.TableName("AIProviderKey") + " AS provider_keys").Joins("JOIN ai_resources ON ai_resources.id = provider_keys.resource_id")
	if text := c.Query("q"); text != "" {
		q = q.Where("provider_keys.name LIKE ? OR ai_resources.name LIKE ?", "%"+text+"%", "%"+text+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		RespondError(c, err)
		return
	}
	items := []struct {
		model.AIProviderKey
		ResourceID      uint   `json:"resourceId"`
		ResourceName    string `json:"resourceName"`
		ResourceEnabled bool   `json:"resourceEnabled"`
	}{}
	if err := q.Select("provider_keys.*, ai_resources.name AS resource_name, ai_resources.enabled AS resource_enabled").Order("provider_keys.id DESC").Offset((page - 1) * size).Limit(size).Scan(&items).Error; err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "pageSize": size})
}

func (h *Handler) SaveAIProviderKey(c *gin.Context) {
	var keyID uint
	if c.Param("id") != "" {
		var ok bool
		keyID, ok = id(c)
		if !ok {
			return
		}
	}
	var input service.AIProviderKeyInput
	if !bind(c, &input) {
		return
	}
	key, err := h.Service.SaveAIProviderKey(keyID, input)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, key)
}

func (h *Handler) DeleteAIProviderKey(c *gin.Context) {
	keyID, ok := id(c)
	if !ok {
		return
	}
	result := h.Service.DB.Delete(&model.AIProviderKey{}, keyID)
	if result.Error != nil {
		RespondError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		RespondError(c, service.Fail(404, "not_found", "供应商密钥不存在"))
		return
	}
	c.Status(204)
}
