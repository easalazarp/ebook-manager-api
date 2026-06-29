package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
)

type CreateAuthorInput struct {
	Name string `json:"name"`
	Bio  string `json:"bio"`
}

type UpdateAuthorInput struct {
	Name *string `json:"name,omitempty"`
	Bio  *string `json:"bio,omitempty"`
}

type IAuthorService interface {
	CreateAuthor(ctx context.Context, input CreateAuthorInput) (*models.Author, error)
	GetAuthor(ctx context.Context, id uuid.UUID) (*models.Author, error)
	ListAuthors(ctx context.Context, p repository.PageParams) (repository.Page[*models.Author], error)
	UpdateAuthor(ctx context.Context, id uuid.UUID, input UpdateAuthorInput) (*models.Author, error)
	DeleteAuthor(ctx context.Context, id uuid.UUID) error
	GetAuthorByBookID(ctx context.Context, bookID uuid.UUID) (*models.Author, error)
}
