package controller

import (
	"encoding/json"
	"net/http"

	"BTDown_MA/internal/common"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

func (controller *HealthController) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, common.SuccessResponse(map[string]string{"status": "UP"}))
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
