package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
)

// IBookRepository define el contrato de acceso a datos para e-books.
type IBookRepository interface {
	Create(ctx context.Context, ebook *models.EBook) (*models.EBook, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.EBook, error)
	List(ctx context.Context, p PageParams) (Page[*models.EBook], error)
	Update(ctx context.Context, ebook *models.EBook) (*models.EBook, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
