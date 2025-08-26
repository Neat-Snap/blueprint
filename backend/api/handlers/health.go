package handlers

import (
	"net/http"

	"github.com/Neat-Snap/blueprint-backend/logger"
	"gorm.io/gorm"
)

type TestHealthAPI struct {
	DB     *gorm.DB
	logger logger.MultiLogger
}

func NewTestHealthAPI(db *gorm.DB, logger logger.MultiLogger) *TestHealthAPI {
	return &TestHealthAPI{DB: db, logger: logger}
}

func (a *TestHealthAPI) HealthHandler(w http.ResponseWriter, r *http.Request) {
	a.logger.Info("health check")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
