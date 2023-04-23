package middleware

import (
	"favor-dao-backend/internal/conf"
	"github.com/gin-gonic/gin"
)

func AllowHost() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.RemoteIP()
		for _, v := range conf.PointSetting.WhiteList {
			if v == host {
				c.Next()
				return
			}
		}
		c.AbortWithStatus(403)
		return
	}
}
