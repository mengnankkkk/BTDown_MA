package controller

import (
	"log"
	"net/http"
	"strings"
	"time"

	"BTDown_MA/internal/service"
)

type StreamController struct {
	streamService      *service.StreamService
	streamAccessBuffer *service.StreamAccessBuffer
	runtimeManager     *service.TorrentRuntimeManager
}

func NewStreamController(streamService *service.StreamService, streamAccessBuffer *service.StreamAccessBuffer, runtimeManager *service.TorrentRuntimeManager) *StreamController {
	return &StreamController{
		streamService:      streamService,
		streamAccessBuffer: streamAccessBuffer,
		runtimeManager:     runtimeManager,
	}
}

func (controller *StreamController) Stream(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	rangeHeader := r.Header.Get("Range")
	userAgent := r.UserAgent()
	writer := &loggedResponseWriter{ResponseWriter: w}

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/streams/")
	if sessionID == "" {
		http.Error(writer, "sessionId 不能为空", http.StatusBadRequest)
		logStreamAccess(startedAt, sessionID, r.Method, rangeHeader, userAgent, writer, controller.streamAccessBuffer, controller.runtimeManager)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		logStreamAccess(startedAt, sessionID, r.Method, rangeHeader, userAgent, writer, controller.streamAccessBuffer, controller.runtimeManager)
		return
	}

	openedStream, err := controller.streamService.OpenStream(r.Context(), sessionID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusConflict)
		logStreamAccess(startedAt, sessionID, r.Method, rangeHeader, userAgent, writer, controller.streamAccessBuffer, controller.runtimeManager)
		return
	}
	defer openedStream.Content.Close()

	http.ServeContent(writer, r, openedStream.Name, openedStream.ModTime, openedStream.Content)
	logStreamAccess(startedAt, sessionID, r.Method, rangeHeader, userAgent, writer, controller.streamAccessBuffer, controller.runtimeManager)
}

type loggedResponseWriter struct {
	http.ResponseWriter
	status int
}

func (writer *loggedResponseWriter) WriteHeader(statusCode int) {
	writer.status = statusCode
	writer.ResponseWriter.WriteHeader(statusCode)
}

func (writer *loggedResponseWriter) statusCode() int {
	if writer.status == 0 {
		return http.StatusOK
	}
	return writer.status
}

func logStreamAccess(startedAt time.Time, sessionID, method, rangeHeader, userAgent string, writer *loggedResponseWriter, streamAccessBuffer *service.StreamAccessBuffer, runtimeManager *service.TorrentRuntimeManager) {
	durationMs := time.Since(startedAt).Milliseconds()
	headers := writer.Header()
	contentRange := headers.Get("Content-Range")
	log.Printf(
		"stream access sessionId=%s method=%s range=%q userAgent=%q status=%d contentRange=%q contentLength=%q acceptRanges=%q durationMs=%d",
		sessionID,
		method,
		rangeHeader,
		userAgent,
		writer.statusCode(),
		contentRange,
		headers.Get("Content-Length"),
		headers.Get("Accept-Ranges"),
		durationMs,
	)
	if streamAccessBuffer != nil {
		streamAccessBuffer.Add(service.StreamAccessRecord{
			At:           time.Now(),
			SessionID:    sessionID,
			Method:       method,
			Range:        rangeHeader,
			Status:       writer.statusCode(),
			DurationMs:   durationMs,
			ContentRange: contentRange,
		})
	}
	if runtimeManager != nil && sessionID != "" {
		runtimeManager.RecordRangeActivity(sessionID, rangeHeader, writer.statusCode(), time.Duration(durationMs)*time.Millisecond)
	}
}
