package service

import (
	"sync"
	"time"
)

const maxStreamAccessRecords = 100

type StreamAccessRecord struct {
	At           time.Time `json:"at"`
	SessionID    string    `json:"sessionId"`
	Method       string    `json:"method"`
	Range        string    `json:"range"`
	Status       int       `json:"status"`
	DurationMs   int64     `json:"durationMs"`
	ContentRange string    `json:"contentRange"`
}

type StreamAccessBuffer struct {
	mutex   sync.RWMutex
	records []StreamAccessRecord
}

func NewStreamAccessBuffer() *StreamAccessBuffer {
	return &StreamAccessBuffer{records: make([]StreamAccessRecord, 0, maxStreamAccessRecords)}
}

func (buffer *StreamAccessBuffer) Add(record StreamAccessRecord) {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()
	if len(buffer.records) == maxStreamAccessRecords {
		buffer.records = append(buffer.records[1:], record)
		return
	}
	buffer.records = append(buffer.records, record)
}

func (buffer *StreamAccessBuffer) ListRecent() []StreamAccessRecord {
	buffer.mutex.RLock()
	defer buffer.mutex.RUnlock()
	result := make([]StreamAccessRecord, len(buffer.records))
	copy(result, buffer.records)
	return result
}
