package http

import (
	"github.com/gin-gonic/gin"
	"minierp/internal/service"
	"minierp/internal/transport/http/handler"
	"minierp/internal/transport/http/middleware"
	"minierp/internal/web"
)

func Router(s *service.Service) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), middleware.Recovery())
	r.SetTrustedProxies(nil)
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "same-origin")
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	})
	h := &handler.Handler{Service: s}
	r.Any("/ai/v1/*path", h.AIProxy)
	api := r.Group("/api")
	api.GET("/auth/status", h.Status)
	api.POST("/auth/bootstrap", h.Bootstrap)
	api.POST("/auth/login", h.Login)
	api.Use(middleware.Auth(s))
	api.GET("/auth/me", h.Me)
	api.POST("/auth/password", h.ChangePassword)
	api.GET("/ai/resources", middleware.RequireSystemAdmin(), h.AIResources)
	api.GET("/ai/resources/options", middleware.RequireSystemAdmin(), h.AIResourceOptions)
	api.POST("/ai/resources", middleware.RequireSystemAdmin(), h.SaveAIResource)
	api.POST("/ai/resources/models", middleware.RequireSystemAdmin(), h.FetchAIModels)
	api.PATCH("/ai/resources/:id", middleware.RequireSystemAdmin(), h.SaveAIResource)
	api.GET("/ai/provider-keys", middleware.RequireSystemAdmin(), h.AIProviderKeys)
	api.POST("/ai/provider-keys", middleware.RequireSystemAdmin(), h.SaveAIProviderKey)
	api.PATCH("/ai/provider-keys/:id", middleware.RequireSystemAdmin(), h.SaveAIProviderKey)
	api.DELETE("/ai/provider-keys/:id", middleware.RequireSystemAdmin(), h.DeleteAIProviderKey)
	api.GET("/ai/models", middleware.RequireSystemAdmin(), h.AIModels)
	api.POST("/ai/models", middleware.RequireSystemAdmin(), h.SaveAIModel)
	api.PATCH("/ai/models/:id", middleware.RequireSystemAdmin(), h.SaveAIModel)
	api.DELETE("/ai/models/:id", middleware.RequireSystemAdmin(), h.DeleteAIModel)
	api.GET("/ai/keys", h.AIKeys)
	api.POST("/ai/keys", middleware.Require("ai:use"), h.CreateAIKey)
	api.POST("/ai/keys/:id/rotate", h.ChangeAIKey)
	api.DELETE("/ai/keys/:id", h.ChangeAIKey)
	api.GET("/system/resources", middleware.RequireSystemAdmin(), h.SystemResources)
	api.GET("/permissions", middleware.Require("roles:read"), h.Permissions)
	api.GET("/users", middleware.Require("users:read"), h.Users)
	api.POST("/users", middleware.Require("users:manage"), h.CreateUser)
	api.GET("/users/:id", middleware.Require("users:read"), h.User)
	api.PATCH("/users/:id", middleware.Require("users:manage"), h.UpdateUser)
	api.PATCH("/users/:id/status", middleware.Require("users:manage"), h.UserStatus)
	api.POST("/users/:id/password-reset", middleware.Require("users:manage"), h.ResetPassword)
	api.GET("/roles", middleware.Require("roles:read", "users:manage"), h.Roles)
	api.GET("/roles/options", middleware.Require("users:manage"), h.RoleOptions)
	api.POST("/roles", middleware.Require("roles:manage"), h.CreateRole)
	api.GET("/roles/:id", middleware.Require("roles:read"), h.Role)
	api.PATCH("/roles/:id", middleware.Require("roles:manage"), h.UpdateRole)
	api.DELETE("/roles/:id", middleware.Require("roles:manage"), h.DeleteRole)
	r.NoRoute(web.Handler())
	return r
}
