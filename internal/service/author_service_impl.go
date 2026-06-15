package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
)

var _ IAuthorService = (*AuthorServiceImpl)(nil)

// AuthorServiceImpl implementa IAuthorService.
type AuthorServiceImpl struct {
	repo repository.IAuthorRepository
}

func NewAuthorServiceImpl(repo repository.IAuthorRepository) *AuthorServiceImpl {
	return &AuthorServiceImpl{repo: repo}
}

func (s *AuthorServiceImpl) CreateAuthor(ctx context.Context, input CreateAuthorInput) (*models.Author, error) {
	now := time.Now().UTC()
	author := &models.Author{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		Name: input.Name,
		Bio:  input.Bio,
	}
	return s.repo.Create(ctx, author)
}

func (s *AuthorServiceImpl) ListAuthors(ctx context.Context) ([]*models.Author, error) {
	return s.repo.ListAll(ctx)
}

func (s *AuthorServiceImpl) GetAuthorByBookID(ctx context.Context, bookID uuid.UUID) (*models.Author, error) {
	return s.repo.GetByBookID(ctx, bookID)
}
