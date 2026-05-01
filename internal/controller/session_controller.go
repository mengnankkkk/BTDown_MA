package controller

import (
	"net/http"
	"strings"

	"BTDown_MA/internal/common"
	"BTDown_MA/internal/dto"
	"BTDown_MA/internal/mapper"
	"BTDown_MA/internal/service"
)

type SessionController struct {
	sessionService *service.SessionService
}

func NewSessionController(sessionService *service.SessionService) *SessionController {
	return &SessionController{sessionService: sessionService}
}

func (controller *SessionController) ListSessions(w http.ResponseWriter, _ *http.Request) {
	sessions := controller.sessionService.ListSessions()
	writeJSON(w, http.StatusOK, common.SuccessResponse(mapper.ToSessionResponseList(sessions)))
}

func (controller *SessionController) CreateSession(w http.ResponseWriter, r *http.Request) {
	var request dto.SessionCreateRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, common.ErrorResponse("请求参数不合法"))
		return
	}
	if strings.TrimSpace(request.MagnetURI) == "" {
		writeJSON(w, http.StatusBadRequest, common.ErrorResponse("magnetUri 不能为空"))
		return
	}

	session := controller.sessionService.CreateSession(request)
	writeJSON(w, http.StatusCreated, common.SuccessResponse(mapper.ToSessionResponse(session)))
}

func (controller *SessionController) StopSession(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/stop"), "/api/v1/sessions/")
	if strings.TrimSpace(sessionID) == "" {
		writeJSON(w, http.StatusBadRequest, common.ErrorResponse("sessionId 不能为空"))
		return
	}
	if err := controller.sessionService.StopSession(sessionID); err != nil {
		writeJSON(w, http.StatusNotFound, common.ErrorResponse(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, common.SuccessResponse(map[string]string{"status": "stopped"}))
}

func (controller *SessionController) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
	if strings.TrimSpace(sessionID) == "" {
		writeJSON(w, http.StatusBadRequest, common.ErrorResponse("sessionId 不能为空"))
		return
	}
	if err := controller.sessionService.DeleteSession(sessionID); err != nil {
		writeJSON(w, http.StatusNotFound, common.ErrorResponse(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, common.SuccessResponse(map[string]string{"status": "deleted"}))
}
