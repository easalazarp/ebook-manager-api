package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
)

// ILogService define la lógica de negocio para registros de auditoría.
type ILogService interface {
	RecordActivity(ctx context.Context, activity string, userID uuid.UUID) error
	GetHistory(ctx context.Context) ([]models.Log, error)
}
