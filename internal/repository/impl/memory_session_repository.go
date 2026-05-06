package impl

import (
	"fmt"
	"sync"

	"BTDown_MA/internal/model"
)

type MemorySessionRepository struct {
	mutex    sync.RWMutex
	sessions map[string]model.Session
}

func NewMemorySessionRepository() *MemorySessionRepository {
	return &MemorySessionRepository{
		sessions: make(map[string]model.Session),
	}
}

func (repository *MemorySessionRepository) Save(session model.Session) model.Session {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	repository.sessions[session.ID] = session
	return session
}

func (repository *MemorySessionRepository) UpdateByID(id string, update func(*model.Session) error) (model.Session, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	return repository.updateLocked(id, update)
}

func (repository *MemorySessionRepository) UpdateByIDTransient(id string, update func(*model.Session) error) (model.Session, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	return repository.updateLocked(id, update)
}

func (repository *MemorySessionRepository) FindAll() []model.Session {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	result := make([]model.Session, 0, len(repository.sessions))
	for _, session := range repository.sessions {
		result = append(result, session)
	}

	return result
}

func (repository *MemorySessionRepository) FindByID(id string) (model.Session, bool) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	session, exists := repository.sessions[id]
	return session, exists
}

func (repository *MemorySessionRepository) DeleteByID(id string) error {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	if _, exists := repository.sessions[id]; !exists {
		return fmt.Errorf("session %s 不存在", id)
	}
	delete(repository.sessions, id)
	return nil
}

func (repository *MemorySessionRepository) Flush() error {
	return nil
}

func (repository *MemorySessionRepository) Close() error {
	return nil
}

func (repository *MemorySessionRepository) updateLocked(id string, update func(*model.Session) error) (model.Session, error) {
	session, exists := repository.sessions[id]
	if !exists {
		return model.Session{}, fmt.Errorf("session %s 不存在", id)
	}
	if err := update(&session); err != nil {
		return model.Session{}, err
	}

	repository.sessions[id] = session
	return session, nil
}
