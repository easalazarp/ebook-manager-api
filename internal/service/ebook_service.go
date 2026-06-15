package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
)

// FileInput representa un archivo a asociar al e-book en la creación.
type FileInput struct {
	Format  string `json:"format"`
	FileURL string `json:"file_url"`
}

// CreateEBookInput contiene los datos necesarios para crear un e-book.
type CreateEBookInput struct {
	AuthorID    uuid.UUID   `json:"author_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Files       []FileInput `json:"files"`
}

// UpdateEBookInput contiene los campos actualizables; nil indica sin cambio.
type UpdateEBookInput struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

// IBookService define la lógica de negocio para e-books.
type IBookService interface {
	CreateEBook(ctx context.Context, input CreateEBookInput) (*models.EBook, error)
	GetEBook(ctx context.Context, id uuid.UUID) (*models.EBook, error)
	ListEBooks(ctx context.Context) ([]*models.EBook, error)
	UpdateEBook(ctx context.Context, id uuid.UUID, input UpdateEBookInput) (*models.EBook, error)
	DeleteEBook(ctx context.Context, id uuid.UUID) error
}
