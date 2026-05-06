package wails

import (
	"BTDown_MA/internal/config"
	"BTDown_MA/internal/dto"
	"BTDown_MA/internal/mapper"
	"BTDown_MA/internal/service"
	"BTDown_MA/internal/vo"
)

type AppBindings struct {
	sessionService       *service.SessionService
	settingsService      *service.SettingsService
	observabilityService *service.ObservabilityService
	playerService        *service.PlayerService
	runtimeManager       *service.TorrentRuntimeManager
}

func NewAppBindings(sessionService *service.SessionService, settingsService *service.SettingsService, observabilityService *service.ObservabilityService, playerService *service.PlayerService, runtimeManager *service.TorrentRuntimeManager) *AppBindings {
	return &AppBindings{
		sessionService:       sessionService,
		settingsService:      settingsService,
		observabilityService: observabilityService,
		playerService:        playerService,
		runtimeManager:       runtimeManager,
	}
}

func (bindings *AppBindings) CreateSession(request dto.SessionCreateRequest) vo.SessionResponse {
	session := bindings.sessionService.CreateSession(request)
	return mapper.ToSessionResponse(session)
}

func (bindings *AppBindings) ListSessions() []vo.SessionResponse {
	return mapper.ToSessionResponseList(bindings.sessionService.ListSessions())
}

func (bindings *AppBindings) GetSettings() config.ApplicationSettings {
	return bindings.settingsService.GetSettings()
}

func (bindings *AppBindings) UpdateSettings(request config.ApplicationSettings) (config.ApplicationSettings, error) {
	updated, err := bindings.settingsService.UpdateSettings(request)
	if err != nil {
		return config.ApplicationSettings{}, err
	}
	bindings.runtimeManager.ApplySettings(updated)
	return updated, nil
}

func (bindings *AppBindings) GetObservabilityOverview() vo.ObservabilityOverviewResponse {
	return bindings.observabilityService.GetOverview()
}

func (bindings *AppBindings) StopSession(sessionID string) error {
	return bindings.sessionService.StopSession(sessionID)
}

func (bindings *AppBindings) PauseSession(sessionID string) error {
	return bindings.sessionService.PauseSession(sessionID)
}

func (bindings *AppBindings) ResumeSession(sessionID string) error {
	return bindings.sessionService.ResumeSession(sessionID)
}

func (bindings *AppBindings) CleanupSession(sessionID string) error {
	return bindings.sessionService.DeleteSession(sessionID)
}

func (bindings *AppBindings) GetPlayerLaunchURL(sessionID string) string {
	return bindings.playerService.BuildPlayerLaunchURL(sessionID)
}
