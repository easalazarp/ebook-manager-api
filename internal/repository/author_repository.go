package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
)

// IAuthorRepository define el contrato de acceso a datos para autores.
type IAuthorRepository interface {
	Create(ctx context.Context, author *models.Author) (*models.Author, error)
	ListAll(ctx context.Context) ([]*models.Author, error)
	GetByBookID(ctx context.Context, bookID uuid.UUID) (*models.Author, error)
}
