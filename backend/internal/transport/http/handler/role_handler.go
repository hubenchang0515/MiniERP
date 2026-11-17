package handler

import (
	"github.com/gin-gonic/gin"
	"minierp/internal/model"
	"minierp/internal/repository"
	"minierp/internal/service"
)

func (h *Handler) Permissions(c *gin.Context) { c.JSON(200, model.Permissions) }
func (h *Handler) Roles(c *gin.Context) {
	page, size, ok := pagination(c)
	if !ok {
		return
	}
	v, total, err := repository.Roles(h.Service.ReadDB().WithContext(c.Request.Context()), c.Query("q"), page, size)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": v, "total": total, "page": page, "pageSize": size})
}
func (h *Handler) RoleOptions(c *gin.Context) {
	page, size, ok := pagination(c)
	if !ok {
		return
	}
	items, total, err := repository.RoleOptions(h.Service.ReadDB().WithContext(c.Request.Context()), c.Query("q"), page, size)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "pageSize": size})
}
func (h *Handler) Role(c *gin.Context) {
	id, ok := id(c)
	if !ok {
		return
	}
	v, err := repository.Role(h.Service.ReadDB().WithContext(c.Request.Context()), id)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, v)
}
func (h *Handler) CreateRole(c *gin.Context) {
	var in service.RoleInput
	if !bind(c, &in) {
		return
	}
	v, err := h.Service.SaveRole(0, in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(201, v)
}
func (h *Handler) UpdateRole(c *gin.Context) {
	id, ok := id(c)
	if !ok {
		return
	}
	var in service.RoleInput
	if !bind(c, &in) {
		return
	}
	v, err := h.Service.SaveRole(id, in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, v)
}
func (h *Handler) DeleteRole(c *gin.Context) {
	id, ok := id(c)
	if !ok {
		return
	}
	if err := h.Service.DeleteRole(id); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
