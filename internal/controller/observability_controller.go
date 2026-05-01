package controller

import (
	"net/http"

	"BTDown_MA/internal/common"
	"BTDown_MA/internal/service"
)

type ObservabilityController struct {
	observabilityService *service.ObservabilityService
}

func NewObservabilityController(observabilityService *service.ObservabilityService) *ObservabilityController {
	return &ObservabilityController{observabilityService: observabilityService}
}

func (controller *ObservabilityController) GetOverview(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, common.SuccessResponse(controller.observabilityService.GetOverview()))
}
