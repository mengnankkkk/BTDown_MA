package impl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"BTDown_MA/internal/model"
)

type FileSessionRepository struct {
	mutex       sync.RWMutex
	sessions    map[string]model.Session
	sessionFile string
}

func NewFileSessionRepository(sessionFile string) (*FileSessionRepository, error) {
	repository := &FileSessionRepository{
		sessions:    make(map[string]model.Session),
		sessionFile: sessionFile,
	}
	if err := repository.load(); err != nil {
		return nil, err
	}
	return repository, nil
}

func (repository *FileSessionRepository) Save(session model.Session) model.Session {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	repository.sessions[session.ID] = session
	if err := repository.persistLocked(); err != nil {
		delete(repository.sessions, session.ID)
		return session
	}
	return session
}

func (repository *FileSessionRepository) UpdateByID(id string, update func(*model.Session) error) (model.Session, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	session, exists := repository.sessions[id]
	if !exists {
		return model.Session{}, fmt.Errorf("session %s 不存在", id)
	}
	if err := update(&session); err != nil {
		return model.Session{}, err
	}

	original := repository.sessions[id]
	repository.sessions[id] = session
	if err := repository.persistLocked(); err != nil {
		repository.sessions[id] = original
		return model.Session{}, err
	}
	return session, nil
}

func (repository *FileSessionRepository) FindAll() []model.Session {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	result := make([]model.Session, 0, len(repository.sessions))
	for _, session := range repository.sessions {
		result = append(result, session)
	}
	return result
}

func (repository *FileSessionRepository) FindByID(id string) (model.Session, bool) {
	repository.mutex.RLock()
	defer repository.mutex.RUnlock()

	session, exists := repository.sessions[id]
	return session, exists
}

func (repository *FileSessionRepository) DeleteByID(id string) error {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	if _, exists := repository.sessions[id]; !exists {
		return fmt.Errorf("session %s 不存在", id)
	}
	original := repository.sessions[id]
	delete(repository.sessions, id)
	if err := repository.persistLocked(); err != nil {
		repository.sessions[id] = original
		return err
	}
	return nil
}

func (repository *FileSessionRepository) load() error {
	payload, err := os.ReadFile(repository.sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("读取会话文件失败: %w", err)
	}
	if len(payload) == 0 {
		return nil
	}

	var sessions []model.Session
	if err := json.Unmarshal(payload, &sessions); err != nil {
		return fmt.Errorf("解析会话文件失败: %w", err)
	}

	loaded := make(map[string]model.Session, len(sessions))
	for _, session := range sessions {
		if session.ID == "" {
			continue
		}
		loaded[session.ID] = session
	}
	repository.sessions = loaded
	return nil
}

func (repository *FileSessionRepository) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(repository.sessionFile), 0o755); err != nil {
		return fmt.Errorf("创建会话目录失败: %w", err)
	}

	sessions := make([]model.Session, 0, len(repository.sessions))
	for _, session := range repository.sessions {
		sessions = append(sessions, session)
	}
	payload, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("编码会话失败: %w", err)
	}
	if err := os.WriteFile(repository.sessionFile, payload, 0o644); err != nil {
		return fmt.Errorf("写入会话文件失败: %w", err)
	}
	return nil
}
