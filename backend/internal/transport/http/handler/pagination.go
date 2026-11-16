package handler

import (
	"github.com/gin-gonic/gin"
	"minierp/internal/service"
	"strconv"
)

func pagination(c *gin.Context) (int, int, bool) {
	page, size := 1, 20
	for _, item := range []struct {
		name   string
		max    int
		target *int
	}{{"page", 1000000, &page}, {"pageSize", 100, &size}} {
		if value, ok := c.GetQuery(item.name); ok {
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > item.max {
				RespondError(c, service.Fail(400, "validation", item.name+" 超出有效范围"))
				return 0, 0, false
			}
			*item.target = n
		}
	}
	return page, size, true
}
