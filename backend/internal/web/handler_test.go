package web

import (
	"github.com/gin-gonic/gin"
	"io/fs"
	"net/http/httptest"
	"testing"
)

func TestStaticAndSPAFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.NoRoute(Handler())
	_, indexErr := fs.Stat(files, "static/index.html")
	for _, path := range []string{"/", "/users/7/edit", "/roles"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		expected := 200
		if indexErr != nil {
			expected = 503
		}
		if w.Code != expected {
			t.Fatalf("%s = %d", path, w.Code)
		}
	}
	for _, path := range []string{"/api/unknown", "/assets/missing.js", "/placeholder.html", "/.keep"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 404 {
			t.Fatalf("%s = %d", path, w.Code)
		}
	}
}
