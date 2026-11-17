package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRecoveryPreservesAbortAndRecoversPanics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name    string
		value   any
		written bool
		abort   bool
	}{
		{"proxy abort", http.ErrAbortHandler, true, true},
		{"ordinary panic", "private-panic-value", false, false},
		{"panic after headers", "private-panic-value", true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			original := gin.DefaultErrorWriter
			gin.DefaultErrorWriter = &logs
			defer func() { gin.DefaultErrorWriter = original }()
			r := gin.New()
			r.Use(Recovery())
			r.GET("/test", func(c *gin.Context) {
				if test.written {
					c.Writer.WriteHeaderNow()
				}
				panic(test.value)
			})
			w := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/test", nil)
			request.Header.Set("Authorization", "Bearer private-token")
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				r.ServeHTTP(w, request)
			}()
			if test.abort && recovered != http.ErrAbortHandler {
				t.Fatalf("expected abort, got %v", recovered)
			}
			if !test.abort && (recovered != nil || w.Code != 500) {
				t.Fatalf("panic=%v status=%d", recovered, w.Code)
			}
			if test.value == http.ErrAbortHandler && logs.Len() != 0 {
				t.Fatalf("abort was logged as panic: %s", &logs)
			}
			if strings.Contains(logs.String(), "private-") {
				t.Fatal("request credential or panic value leaked")
			}
		})
	}
}
