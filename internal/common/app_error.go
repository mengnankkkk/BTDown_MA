package common

type AppError struct {
	Message string
}

func (appError *AppError) Error() string {
	return appError.Message
}

func NewAppError(message string) error {
	return &AppError{Message: message}
}
