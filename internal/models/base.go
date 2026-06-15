package models

import (
	"time"

	"github.com/google/uuid"
)

// BaseModel contiene los campos de auditoría comunes a todas las entidades.
// Se embebe en cada modelo de dominio.
type BaseModel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
