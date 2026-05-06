package impl

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"BTDown_MA/internal/model"
)

const sessionFlushInterval = 1500 * time.Millisecond

type FileSessionRepository struct {
	mutex         sync.RWMutex
	sessions      map[string]model.Session
	sessionFile   string
	dirty         bool
	closed        bool
	flushSignal   chan struct{}
	stopCh        chan struct{}
	doneCh        chan struct{}
	flushInterval time.Duration
}

func NewFileSessionRepository(sessionFile string) (*FileSessionRepository, error) {
	repository := &FileSessionRepository{
		sessions:      make(map[string]model.Session),
		sessionFile:   sessionFile,
		flushSignal:   make(chan struct{}, 1),
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
		flushInterval: sessionFlushInterval,
	}
	if err := repository.load(); err != nil {
		return nil, err
	}
	go repository.flushLoop()
	return repository, nil
}

func (repository *FileSessionRepository) Save(session model.Session) model.Session {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	original, existed := repository.sessions[session.ID]
	previousDirty := repository.dirty
	repository.sessions[session.ID] = session
	repository.dirty = true
	if err := repository.persistLocked(); err != nil {
		if existed {
			repository.sessions[session.ID] = original
		} else {
			delete(repository.sessions, session.ID)
		}
		repository.dirty = previousDirty
		return session
	}
	return session
}

func (repository *FileSessionRepository) UpdateByID(id string, update func(*model.Session) error) (model.Session, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	return repository.updateLocked(id, update, true)
}

func (repository *FileSessionRepository) UpdateByIDTransient(id string, update func(*model.Session) error) (model.Session, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	return repository.updateLocked(id, update, false)
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
	previousDirty := repository.dirty
	delete(repository.sessions, id)
	repository.dirty = true
	if err := repository.persistLocked(); err != nil {
		repository.sessions[id] = original
		repository.dirty = previousDirty
		return err
	}
	return nil
}

func (repository *FileSessionRepository) Flush() error {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	return repository.persistIfDirtyLocked()
}

func (repository *FileSessionRepository) Close() error {
	repository.mutex.Lock()
	if repository.closed {
		repository.mutex.Unlock()
		return nil
	}
	repository.closed = true
	close(repository.stopCh)
	repository.mutex.Unlock()
	<-repository.doneCh
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

func (repository *FileSessionRepository) flushLoop() {
	ticker := time.NewTicker(repository.flushInterval)
	defer ticker.Stop()
	defer close(repository.doneCh)

	for {
		select {
		case <-ticker.C:
			repository.flushIfDirtyAsync()
		case <-repository.flushSignal:
			repository.flushIfDirtyAsync()
		case <-repository.stopCh:
			repository.flushIfDirtyAsync()
			return
		}
	}
}

func (repository *FileSessionRepository) flushIfDirtyAsync() {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	if err := repository.persistIfDirtyLocked(); err != nil {
		log.Printf("flush sessions failed: %v", err)
	}
}

func (repository *FileSessionRepository) persistIfDirtyLocked() error {
	if !repository.dirty {
		return nil
	}
	return repository.persistLocked()
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
	repository.dirty = false
	return nil
}

func (repository *FileSessionRepository) updateLocked(id string, update func(*model.Session) error, requestFlush bool) (model.Session, error) {
	session, exists := repository.sessions[id]
	if !exists {
		return model.Session{}, fmt.Errorf("session %s 不存在", id)
	}
	if err := update(&session); err != nil {
		return model.Session{}, err
	}

	original := repository.sessions[id]
	if reflect.DeepEqual(session, original) {
		return session, nil
	}
	repository.sessions[id] = session
	repository.dirty = true
	if requestFlush {
		repository.requestFlushLocked()
	}
	return session, nil
}

func (repository *FileSessionRepository) requestFlushLocked() {
	select {
	case repository.flushSignal <- struct{}{}:
	default:
	}
}
