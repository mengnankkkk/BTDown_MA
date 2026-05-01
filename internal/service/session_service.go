package service

import (
	"fmt"
	"time"

	"BTDown_MA/internal/dto"
	"BTDown_MA/internal/model"
	"BTDown_MA/internal/repository"
)

type SessionService struct {
	sessionRepository repository.SessionRepository
	streamService     *StreamService
	runtimeManager    *torrentRuntimeManager
}

func NewSessionService(
	sessionRepository repository.SessionRepository,
	streamService *StreamService,
	runtimeManager *torrentRuntimeManager,
) *SessionService {
	return &SessionService{
		sessionRepository: sessionRepository,
		streamService:     streamService,
		runtimeManager:    runtimeManager,
	}
}

func (service *SessionService) CreateSession(request dto.SessionCreateRequest) model.Session {
	now := time.Now()
	sessionID := fmt.Sprintf("session-%d", now.UnixNano())
	name := request.Name
	if name == "" {
		name = "未命名会话"
	}

	session := model.Session{
		ID:            sessionID,
		MagnetURI:     request.MagnetURI,
		Name:          name,
		Status:        model.SessionStatusInitializing,
		MetadataState: model.SessionMetadataStatePending,
		DownloadState: model.SessionDownloadStateQueued,
		StreamState:   model.SessionStreamStateUnavailable,
		DeadState:     model.SessionDeadStateUnknown,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	session.StreamURL = service.streamService.BuildStreamURL(session.ID)
	refreshSessionDerivedFields(&session, 0)

	savedSession := service.sessionRepository.Save(session)
	service.runtimeManager.startSession(savedSession)
	return savedSession
}

func (service *SessionService) ListSessions() []model.Session {
	return service.sessionRepository.FindAll()
}

func (service *SessionService) StopSession(id string) error {
	if _, exists := service.sessionRepository.FindByID(id); !exists {
		return fmt.Errorf("session %s 不存在", id)
	}
	return service.runtimeManager.stopSession(id)
}

func (service *SessionService) DeleteSession(id string) error {
	if _, exists := service.sessionRepository.FindByID(id); !exists {
		return fmt.Errorf("session %s 不存在", id)
	}
	if err := service.runtimeManager.cleanupSession(id); err != nil {
		return err
	}
	return service.sessionRepository.DeleteByID(id)
}
