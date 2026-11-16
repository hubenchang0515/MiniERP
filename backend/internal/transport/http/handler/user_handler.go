package handler

import (
	"github.com/gin-gonic/gin"
	"minierp/internal/repository"
	"minierp/internal/service"
	"strconv"
)

func (h *Handler) Users(c *gin.Context) {
	page, size := 1, 20
	var err error
	if v, ok := c.GetQuery("page"); ok {
		page, err = strconv.Atoi(v)
		if err != nil || page < 1 || page > 1000000 {
			RespondError(c, service.Fail(400, "validation", "page 必须为 1–1000000"))
			return
		}
	}
	if v, ok := c.GetQuery("pageSize"); ok {
		size, err = strconv.Atoi(v)
		if err != nil || size < 1 || size > 100 {
			RespondError(c, service.Fail(400, "validation", "pageSize 必须为 1–100"))
			return
		}
	}
	users, total, err := repository.Users(h.Service.DB, c.Query("q"), page, size)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": users, "total": total, "page": page, "pageSize": size})
}
func (h *Handler) User(c *gin.Context) {
	id, ok := id(c)
	if !ok {
		return
	}
	v, err := repository.User(h.Service.DB, id)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, v)
}
func (h *Handler) CreateUser(c *gin.Context) {
	var in service.UserInput
	if !bind(c, &in) {
		return
	}
	v, err := h.Service.CreateUser(in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(201, v)
}
func (h *Handler) UpdateUser(c *gin.Context) {
	id, ok := id(c)
	if !ok {
		return
	}
	var in service.UserInput
	if !bind(c, &in) {
		return
	}
	v, err := h.Service.UpdateUser(id, in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, v)
}
func (h *Handler) UserStatus(c *gin.Context) {
	id, ok := id(c)
	if !ok {
		return
	}
	var in struct {
		Enabled *bool `json:"enabled"`
	}
	if !bind(c, &in) {
		return
	}
	if in.Enabled == nil {
		RespondError(c, service.Fail(400, "validation", "必须提供 enabled"))
		return
	}
	if err := h.Service.SetStatus(id, *in.Enabled); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
func (h *Handler) ResetPassword(c *gin.Context) {
	id, ok := id(c)
	if !ok {
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if !bind(c, &in) {
		return
	}
	if err := h.Service.ResetPassword(id, in.Password); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
