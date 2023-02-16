//go:build docs
// +build docs

package routers

import (
	"favor-dao-backend/docs"
	"github.com/gin-gonic/gin"
)

// registerDocs register docs asset route
func registerDocs(e *gin.Engine) {
	e.StaticFS("/docs", docs.NewFileSystem())
}
