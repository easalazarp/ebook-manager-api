package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
)

// CreateAuthorInput contiene los datos para crear un autor.
type CreateAuthorInput struct {
	Name string `json:"name"`
	Bio  string `json:"bio"`
}

// IAuthorService define la lógica de negocio para autores.
type IAuthorService interface {
	CreateAuthor(ctx context.Context, input CreateAuthorInput) (*models.Author, error)
	ListAuthors(ctx context.Context) ([]*models.Author, error)
	GetAuthorByBookID(ctx context.Context, bookID uuid.UUID) (*models.Author, error)
}
