package service

type PlayerService struct {
	streamService *StreamService
}

func NewPlayerService(streamService *StreamService) *PlayerService {
	return &PlayerService{streamService: streamService}
}

func (service *PlayerService) BuildPlayerLaunchURL(sessionID string) string {
	return service.streamService.BuildStreamURL(sessionID)
}
