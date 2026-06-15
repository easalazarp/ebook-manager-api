package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
)

var _ ILogService = (*LogServiceImpl)(nil)

// LogServiceImpl implementa ILogService.
type LogServiceImpl struct {
	repo repository.ILogRepository
}

func NewLogServiceImpl(repo repository.ILogRepository) *LogServiceImpl {
	return &LogServiceImpl{repo: repo}
}

func (s *LogServiceImpl) RecordActivity(ctx context.Context, activity string, userID uuid.UUID) error {
	log := &models.Log{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		Activity: activity,
		UserID:   userID,
	}
	return s.repo.SaveLog(ctx, log)
}

func (s *LogServiceImpl) GetHistory(ctx context.Context) ([]models.Log, error) {
	return s.repo.GetHistory(ctx)
}
