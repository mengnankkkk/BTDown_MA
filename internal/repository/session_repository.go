package repository

import (
	"BTDown_MA/internal/model"
)

type SessionRepository interface {
	Save(session model.Session) model.Session
	UpdateByID(id string, update func(*model.Session) error) (model.Session, error)
	UpdateByIDTransient(id string, update func(*model.Session) error) (model.Session, error)
	FindAll() []model.Session
	FindByID(id string) (model.Session, bool)
	DeleteByID(id string) error
	Flush() error
	Close() error
}
