package internal

import (
	"favor-dao-backend/internal/routers/api"
	"favor-dao-backend/internal/service"
)

func Initialize() {
	// initialize service
	service.Initialize()
	api.Initialize()
}
