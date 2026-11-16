package handler

import "github.com/gin-gonic/gin"

func (h *Handler) SystemResources(c *gin.Context) {
	result, err := h.Service.SystemResources()
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, result)
}
