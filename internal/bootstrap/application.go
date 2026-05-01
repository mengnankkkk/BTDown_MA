package bootstrap

import (
	"fmt"
	"net/http"

	"BTDown_MA/internal/config"
	"BTDown_MA/internal/controller"
	"BTDown_MA/internal/repository/impl"
	"BTDown_MA/internal/service"
	"BTDown_MA/internal/stream"
	"BTDown_MA/internal/wails"
)

type Application struct {
	config       config.ApplicationConfig
	httpServer   *http.Server
	wailsBinding *wails.AppBindings
}

func NewApplication() *Application {
	applicationConfig := config.LoadApplicationConfig()
	settingsService := service.NewSettingsService(applicationConfig.SettingsFile, applicationConfig.Settings)
	currentSettings := settingsService.GetSettings()

	sessionRepository := impl.NewMemorySessionRepository()
	runtimeManager, err := service.NewTorrentRuntimeManager(sessionRepository, currentSettings)
	if err != nil {
		panic(err)
	}
	streamService := service.NewStreamService(currentSettings.StreamBaseURL, runtimeManager)
	sessionService := service.NewSessionService(sessionRepository, streamService, runtimeManager)
	streamAccessBuffer := service.NewStreamAccessBuffer()
	observabilityService := service.NewObservabilityService(sessionService, streamAccessBuffer)
	playerService := service.NewPlayerService(streamService)

	healthController := controller.NewHealthController()
	sessionController := controller.NewSessionController(sessionService)
	settingsController := controller.NewSettingsController(settingsService)
	observabilityController := controller.NewObservabilityController(observabilityService)
	streamController := controller.NewStreamController(streamService, streamAccessBuffer, runtimeManager)
	streamServer := stream.NewHTTPStreamServer(
		applicationConfig.ServerAddress,
		healthController,
		sessionController,
		settingsController,
		observabilityController,
		streamController,
	)

	return &Application{
		config:       applicationConfig,
		httpServer:   streamServer.BuildServer(),
		wailsBinding: wails.NewAppBindings(sessionService, settingsService, observabilityService, playerService),
	}
}

func (application *Application) Run() error {
	fmt.Printf("BTDown_MA backend listening on %s\n", application.config.ServerAddress)
	_ = application.wailsBinding
	return application.httpServer.ListenAndServe()
}
