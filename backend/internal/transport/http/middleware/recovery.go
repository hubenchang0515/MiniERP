package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery preserves net/http's abort signal so an incomplete proxy response
// closes the connection (or resets the HTTP/2 stream) instead of ending normally.
func Recovery() gin.HandlerFunc {
	logger := log.New(gin.DefaultErrorWriter, "", log.LstdFlags)
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if recovered == http.ErrAbortHandler {
					panic(http.ErrAbortHandler)
				}
				// Do not dump request headers, bodies, or panic values containing credentials.
				logger.Printf("handler panic method=%s route=%s type=%T\n%s", c.Request.Method, c.FullPath(), recovered, debug.Stack())
				if c.Writer.Written() {
					panic(http.ErrAbortHandler)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
