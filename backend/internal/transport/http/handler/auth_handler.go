package handler

import (
	"github.com/gin-gonic/gin"
	"minioa/internal/model"
	"minioa/internal/service"
)

func (h *Handler) Status(c *gin.Context) {
	v, err := h.Service.Initialized()
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, gin.H{"initialized": v})
}
func (h *Handler) Bootstrap(c *gin.Context) {
	var in service.Credentials
	if !bind(c, &in) {
		return
	}
	v, err := h.Service.Bootstrap(in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(201, v)
}
func (h *Handler) Login(c *gin.Context) {
	var in service.Credentials
	if !bind(c, &in) {
		return
	}
	v, err := h.Service.Login(in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, v)
}
func (h *Handler) Me(c *gin.Context) { c.JSON(200, service.Effective(c.MustGet("user").(model.User))) }

func (h *Handler) ChangePassword(c *gin.Context) {
	var in service.ChangePasswordInput
	if !bind(c, &in) {
		return
	}
	u := c.MustGet("user").(model.User)
	if err := h.Service.ChangePassword(u.ID, in); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(204)
}
