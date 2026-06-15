package models

import "github.com/google/uuid"

// Log registra una actividad de auditoría del sistema.
type Log struct {
	BaseModel
	Activity string    `json:"activity"`
	UserID   uuid.UUID `json:"user_id"`
}

// GetUser devuelve el UUID del usuario que generó la actividad.
func (l *Log) GetUser() uuid.UUID { return l.UserID }
