package web

import (
	"github.com/gin-gonic/gin"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

func Handler() gin.HandlerFunc {
	sub, _ := fs.Sub(files, "static")
	server := http.FileServer(http.FS(sub))
	return func(c *gin.Context) {
		p := path.Clean("/" + c.Request.URL.Path)
		if p == "/api" || strings.HasPrefix(p, "/api/") {
			c.JSON(404, gin.H{"code": "not_found", "message": "接口不存在"})
			return
		}
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
			c.Status(405)
			return
		}
		if strings.Contains(p, "/.") || p == "/placeholder.html" {
			c.Status(404)
			return
		}
		name := strings.TrimPrefix(p, "/")
		if name != "" {
			if info, err := fs.Stat(sub, name); err == nil && !info.IsDir() {
				if strings.HasPrefix(p, "/assets/") {
					c.Header("Cache-Control", "public, max-age=31536000, immutable")
				}
				server.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		if path.Ext(p) != "" {
			c.Status(404)
			return
		}
		b, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			c.Data(503, "text/html; charset=utf-8", []byte("请先运行构建脚本生成前端资源"))
			return
		}
		c.Header("Cache-Control", "no-cache")
		if c.Request.Method == "HEAD" {
			c.Status(200)
			return
		}
		c.Data(200, "text/html; charset=utf-8", b)
	}
}
