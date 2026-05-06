package controller

import (
	"net/http"

	"BTDown_MA/internal/common"
	"BTDown_MA/internal/config"
	"BTDown_MA/internal/service"
)

type SettingsController struct {
	settingsService *service.SettingsService
	runtimeManager  *service.TorrentRuntimeManager
}

func NewSettingsController(settingsService *service.SettingsService, runtimeManager *service.TorrentRuntimeManager) *SettingsController {
	return &SettingsController{settingsService: settingsService, runtimeManager: runtimeManager}
}

func (controller *SettingsController) GetSettings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, common.SuccessResponse(controller.settingsService.GetSettings()))
}

func (controller *SettingsController) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var request config.ApplicationSettings
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, common.ErrorResponse("请求参数不合法"))
		return
	}

	updated, err := controller.settingsService.UpdateSettings(request)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, common.ErrorResponse(err.Error()))
		return
	}
	controller.runtimeManager.ApplySettings(updated)
	writeJSON(w, http.StatusOK, common.SuccessResponse(updated))
}
