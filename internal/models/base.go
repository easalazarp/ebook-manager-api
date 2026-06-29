package models

import (
	"time"

	"github.com/google/uuid"
)

// BaseModel contiene los campos de auditoría comunes a todas las entidades.
type BaseModel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// newUUID y timeNow son helpers internos para construir entidades en los modelos.
func newUUID() uuid.UUID { return uuid.New() }
func timeNow() time.Time { return time.Now().UTC() }
