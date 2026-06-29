package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
)

var _ IAuthorService = (*AuthorServiceImpl)(nil)

type AuthorServiceImpl struct {
	repo repository.IAuthorRepository
}

func NewAuthorServiceImpl(repo repository.IAuthorRepository) *AuthorServiceImpl {
	return &AuthorServiceImpl{repo: repo}
}

func (s *AuthorServiceImpl) CreateAuthor(ctx context.Context, input CreateAuthorInput) (*models.Author, error) {
	now := time.Now().UTC()
	author := &models.Author{
		BaseModel: models.BaseModel{ID: uuid.New(), CreatedAt: now, UpdatedAt: now},
		Name:      input.Name,
		Bio:       input.Bio,
	}
	return s.repo.Create(ctx, author)
}

func (s *AuthorServiceImpl) GetAuthor(ctx context.Context, id uuid.UUID) (*models.Author, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *AuthorServiceImpl) ListAuthors(ctx context.Context, p repository.PageParams) (repository.Page[*models.Author], error) {
	return s.repo.ListAll(ctx, p)
}

func (s *AuthorServiceImpl) UpdateAuthor(ctx context.Context, id uuid.UUID, input UpdateAuthorInput) (*models.Author, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		existing.Name = *input.Name
	}
	if input.Bio != nil {
		existing.Bio = *input.Bio
	}
	return s.repo.Update(ctx, existing)
}

func (s *AuthorServiceImpl) DeleteAuthor(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *AuthorServiceImpl) GetAuthorByBookID(ctx context.Context, bookID uuid.UUID) (*models.Author, error) {
	return s.repo.GetByBookID(ctx, bookID)
}
