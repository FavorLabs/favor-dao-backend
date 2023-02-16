package api

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/dao"
)

var (
	objectStorage core.ObjectStorageService
)

func Initialize() {
	objectStorage = dao.ObjectStorageService()
}
