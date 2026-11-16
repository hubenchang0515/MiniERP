package handler

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"minierp/internal/service"
	"net/http"
	"strconv"
)

type Handler struct{ Service *service.Service }

func RespondError(c *gin.Context, err error) {
	err = service.Normalize(err)
	var e *service.Error
	if errors.As(err, &e) {
		c.AbortWithStatusJSON(e.Status, gin.H{"code": e.Code, "message": e.Message})
		return
	}
	log.Printf("request %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	c.AbortWithStatusJSON(500, gin.H{"code": "internal_error", "message": "服务器处理失败，请稍后重试"})
}
func bind(c *gin.Context, v any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		RespondError(c, service.Fail(400, "invalid_json", "请求 JSON 格式不正确"))
		return false
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		RespondError(c, service.Fail(400, "invalid_json", "请求只接受单个 JSON 对象"))
		return false
	}
	return true
}
func id(c *gin.Context) (uint, bool) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || n == 0 {
		RespondError(c, service.Fail(400, "invalid_id", "ID 必须是正整数"))
		return 0, false
	}
	return uint(n), true
}
