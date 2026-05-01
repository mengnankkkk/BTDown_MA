package service

import (
	"context"
	"io"
	"time"

	"BTDown_MA/internal/common"
)

type StreamService struct {
	streamBaseURL  string
	runtimeManager *torrentRuntimeManager
}

type OpenedStream struct {
	Name    string
	ModTime time.Time
	Size    int64
	Content io.ReadSeekCloser
}

func NewStreamService(streamBaseURL string, runtimeManager *torrentRuntimeManager) *StreamService {
	return &StreamService{
		streamBaseURL:  streamBaseURL,
		runtimeManager: runtimeManager,
	}
}

func (service *StreamService) BuildStreamURL(sessionID string) string {
	return service.streamBaseURL + "/api/v1/streams/" + sessionID
}

func (service *StreamService) OpenStream(ctx context.Context, sessionID string) (*OpenedStream, error) {
	resource, err := service.runtimeManager.openStream(ctx, sessionID)
	if err != nil {
		return nil, common.NewAppError(err.Error())
	}

	return &OpenedStream{
		Name:    resource.name,
		ModTime: resource.modTime,
		Size:    resource.size,
		Content: resource.content,
	}, nil
}
