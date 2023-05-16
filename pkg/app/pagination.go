package app

import (
	"time"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/pkg/convert"
	"github.com/gin-gonic/gin"
)

func GetPage(c *gin.Context) int {
	page := convert.StrTo(c.Query("page")).MustInt()
	if page <= 0 {
		return 1
	}

	return page
}

func GetPageSize(c *gin.Context) int {
	pageSize := convert.StrTo(c.Query("page_size")).MustInt()
	if pageSize <= 0 {
		return conf.AppSetting.DefaultPageSize
	}
	if pageSize > conf.AppSetting.MaxPageSize {
		return conf.AppSetting.MaxPageSize
	}

	return pageSize
}

func GetPageOffset(c *gin.Context) (offset, limit int) {
	page := convert.StrTo(c.Query("page")).MustInt()
	if page <= 0 {
		page = 1
	}

	limit = convert.StrTo(c.Query("page_size")).MustInt()
	if limit <= 0 {
		limit = conf.AppSetting.DefaultPageSize
	} else if limit > conf.AppSetting.MaxPageSize {
		limit = conf.AppSetting.MaxPageSize
	}
	offset = (page - 1) * limit
	return
}

func GetYear(c *gin.Context) (start, end int64) {
	y := convert.StrTo(c.Query("year")).MustInt()
	start = time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	end = time.Date(y+1, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	return
}
