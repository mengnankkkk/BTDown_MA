package dto

type SessionCreateRequest struct {
	MagnetURI string `json:"magnetUri"`
	Name      string `json:"name"`
}
