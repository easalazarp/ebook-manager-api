package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
)

// ICategoryRepository define el contrato de acceso a datos para categorías.
type ICategoryRepository interface {
	Create(ctx context.Context, category *models.Category) (*models.Category, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Category, error)
	ListAll(ctx context.Context, p PageParams) (Page[*models.Category], error)
	Update(ctx context.Context, category *models.Category) (*models.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
