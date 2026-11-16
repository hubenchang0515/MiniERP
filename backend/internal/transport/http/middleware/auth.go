package middleware

import (
	"github.com/gin-gonic/gin"
	"minierp/internal/model"
	"minierp/internal/service"
	"minierp/internal/transport/http/handler"
	"strings"
)

func Auth(s *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			handler.RespondError(c, service.Fail(401, "unauthorized", "请先登录"))
			return
		}
		u, err := s.Authenticate(parts[1])
		if err != nil {
			handler.RespondError(c, err)
			return
		}
		c.Set("user", u)
		c.Next()
	}
}
func Require(codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := c.MustGet("user").(model.User)
		for _, code := range codes {
			if service.Can(u, code) {
				c.Next()
				return
			}
		}
		handler.RespondError(c, service.Fail(403, "forbidden", "没有执行此操作的权限"))
	}
}

func RequireSystemAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := c.MustGet("user").(model.User)
		for _, role := range u.Roles {
			if role.System {
				c.Next()
				return
			}
		}
		handler.RespondError(c, service.Fail(403, "forbidden", "仅系统管理员可查看系统资源"))
	}
}
