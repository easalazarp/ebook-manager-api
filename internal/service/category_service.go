package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
)

type CreateCategoryInput struct {
	Name string `json:"name"`
}

type UpdateCategoryInput struct {
	Name *string `json:"name,omitempty"`
}

type ICategoryService interface {
	CreateCategory(ctx context.Context, input CreateCategoryInput) (*models.Category, error)
	GetCategory(ctx context.Context, id uuid.UUID) (*models.Category, error)
	ListCategories(ctx context.Context, p repository.PageParams) (repository.Page[*models.Category], error)
	UpdateCategory(ctx context.Context, id uuid.UUID, input UpdateCategoryInput) (*models.Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}
