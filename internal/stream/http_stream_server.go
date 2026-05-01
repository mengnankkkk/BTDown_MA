package stream

import (
	"net/http"

	"BTDown_MA/internal/controller"
)

type HTTPStreamServer struct {
	serverAddress          string
	healthController       *controller.HealthController
	sessionController      *controller.SessionController
	settingsController     *controller.SettingsController
	observabilityController *controller.ObservabilityController
	streamController       *controller.StreamController
}

func NewHTTPStreamServer(
	serverAddress string,
	healthController *controller.HealthController,
	sessionController *controller.SessionController,
	settingsController *controller.SettingsController,
	observabilityController *controller.ObservabilityController,
	streamController *controller.StreamController,
) *HTTPStreamServer {
	return &HTTPStreamServer{
		serverAddress:           serverAddress,
		healthController:        healthController,
		sessionController:       sessionController,
		settingsController:      settingsController,
		observabilityController: observabilityController,
		streamController:        streamController,
	}
}

func (server *HTTPStreamServer) BuildServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", server.handleHealth)
	mux.HandleFunc("/api/v1/sessions", server.handleSessions)
	mux.HandleFunc("/api/v1/sessions/", server.handleSessionByID)
	mux.HandleFunc("/api/v1/settings", server.handleSettings)
	mux.HandleFunc("/api/v1/observability/overview", server.handleObservabilityOverview)
	mux.HandleFunc("/api/v1/streams/", server.handleStream)

	return &http.Server{
		Addr:    server.serverAddress,
		Handler: mux,
	}
}

func (server *HTTPStreamServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if handlePreflight(w, r) {
		return
	}
	applyCORSHeaders(w)
	server.healthController.Health(w, r)
}

func (server *HTTPStreamServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	if handlePreflight(w, r) {
		return
	}
	applyCORSHeaders(w)

	switch r.Method {
	case http.MethodGet:
		server.sessionController.ListSessions(w, r)
	case http.MethodPost:
		server.sessionController.CreateSession(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (server *HTTPStreamServer) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	if handlePreflight(w, r) {
		return
	}
	applyCORSHeaders(w)

	switch {
	case r.Method == http.MethodPost && len(r.URL.Path) > len("/api/v1/sessions/") && r.URL.Path[len(r.URL.Path)-5:] == "/stop":
		server.sessionController.StopSession(w, r)
	case r.Method == http.MethodDelete:
		server.sessionController.DeleteSession(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (server *HTTPStreamServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	if handlePreflight(w, r) {
		return
	}
	applyCORSHeaders(w)

	switch r.Method {
	case http.MethodGet:
		server.settingsController.GetSettings(w, r)
	case http.MethodPut:
		server.settingsController.UpdateSettings(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (server *HTTPStreamServer) handleObservabilityOverview(w http.ResponseWriter, r *http.Request) {
	if handlePreflight(w, r) {
		return
	}
	applyCORSHeaders(w)

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	server.observabilityController.GetOverview(w, r)
}

func (server *HTTPStreamServer) handleStream(w http.ResponseWriter, r *http.Request) {
	if handlePreflight(w, r) {
		return
	}
	applyCORSHeaders(w)
	server.streamController.Stream(w, r)
}
