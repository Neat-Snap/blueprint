package utils

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Neat-Snap/blueprint-backend/logger"
)

type DefaultResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type UserConciseResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func ReadJSON(b io.ReadCloser, w http.ResponseWriter, logger logger.MultiLogger, v any) error {
	defer b.Close()
	err := json.NewDecoder(b).Decode(v)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logger.Warn("failed to decode request body", "error", err)
		if err = json.NewEncoder(w).Encode(DefaultResponse{
			Message: "Failed to decode request body",
			Success: false,
		}); err != nil {
			logger.Error("failed to encode response", "error", err)
		}
		return err
	}
	return nil
}

func WriteError(w http.ResponseWriter, logger logger.MultiLogger, err error, message string, status int) {
	w.WriteHeader(status)
	logger.Warn(message, "error", err, "status", status)
	if err := json.NewEncoder(w).Encode(DefaultResponse{
		Message: message,
		Success: false,
	}); err != nil {
		logger.Error("failed to encode response", "error", err)
	}
}

func WriteSuccess(w http.ResponseWriter, logger logger.MultiLogger, obj any, status int) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(obj); err != nil {
		logger.Error("failed to encode response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func PickNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}
