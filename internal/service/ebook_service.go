package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
)

type FileInput struct {
	Format  string `json:"format"`
	FileURL string `json:"file_url"`
}

type CreateEBookInput struct {
	AuthorID    uuid.UUID   `json:"author_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	CoverURL    string      `json:"cover_url,omitempty"`
	Files       []FileInput `json:"files"`
}

// UpdateEBookInput — todos los campos son punteros: nil = no cambiar.
type UpdateEBookInput struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	AuthorID    *uuid.UUID `json:"author_id,omitempty"`
	CoverURL    *string    `json:"cover_url,omitempty"`
}

type IBookService interface {
	CreateEBook(ctx context.Context, input CreateEBookInput) (*models.EBook, error)
	GetEBook(ctx context.Context, id uuid.UUID) (*models.EBook, error)
	ListEBooks(ctx context.Context, p repository.PageParams) (repository.Page[*models.EBook], error)
	UpdateEBook(ctx context.Context, id uuid.UUID, input UpdateEBookInput) (*models.EBook, error)
	DeleteEBook(ctx context.Context, id uuid.UUID) error
}
