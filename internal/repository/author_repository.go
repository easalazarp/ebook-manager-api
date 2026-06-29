package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
)

// IAuthorRepository define el contrato de acceso a datos para autores.
type IAuthorRepository interface {
	Create(ctx context.Context, author *models.Author) (*models.Author, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Author, error)
	ListAll(ctx context.Context, p PageParams) (Page[*models.Author], error)
	Update(ctx context.Context, author *models.Author) (*models.Author, error)
	Delete(ctx context.Context, id uuid.UUID) error
	GetByBookID(ctx context.Context, bookID uuid.UUID) (*models.Author, error)
}
