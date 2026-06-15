package repository

import (
	"context"

	"github.com/yourorg/ebook-management-backend/internal/models"
)

// ILogRepository define el contrato de acceso a datos para logs de auditoría.
type ILogRepository interface {
	SaveLog(ctx context.Context, log *models.Log) error
	GetHistory(ctx context.Context) ([]models.Log, error)
}
