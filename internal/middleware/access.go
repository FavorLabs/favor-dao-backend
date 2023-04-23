package middleware

import (
	"github.com/gin-gonic/gin"
)

var allowHost = []string{
	"localhost",
	"127.0.0.1",
}

func AllowHost() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.RemoteIP()
		for _, v := range allowHost {
			if v == host {
				c.Next()
				return
			}
		}
		c.Abort()
		return
	}
}
