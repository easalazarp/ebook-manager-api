package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
)

var _ IBookService = (*EBookServiceImpl)(nil)

type EBookServiceImpl struct {
	repo       repository.IBookRepository
	logService ILogService
}

func NewEBookServiceImpl(repo repository.IBookRepository, logService ILogService) *EBookServiceImpl {
	return &EBookServiceImpl{repo: repo, logService: logService}
}

func (s *EBookServiceImpl) CreateEBook(ctx context.Context, input CreateEBookInput) (*models.EBook, error) {
	for _, f := range input.Files {
		if err := models.ValidateFormat(f.Format); err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC()
	ebook := &models.EBook{
		BaseModel:   models.BaseModel{ID: uuid.New(), CreatedAt: now, UpdatedAt: now},
		AuthorID:    input.AuthorID,
		Title:       input.Title,
		Description: input.Description,
		CoverURL:    input.CoverURL,
		Files:       make([]models.EBookFile, 0, len(input.Files)),
	}
	for _, f := range input.Files {
		ebook.Files = append(ebook.Files, models.EBookFile{ID: uuid.New(), Format: f.Format, FileURL: f.FileURL})
	}
	created, err := s.repo.Create(ctx, ebook)
	if err != nil {
		return nil, err
	}
	_ = s.logService.RecordActivity(ctx, "ebook_created", uuid.Nil)
	return created, nil
}

func (s *EBookServiceImpl) GetEBook(ctx context.Context, id uuid.UUID) (*models.EBook, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *EBookServiceImpl) ListEBooks(ctx context.Context, p repository.PageParams) (repository.Page[*models.EBook], error) {
	return s.repo.List(ctx, p)
}

func (s *EBookServiceImpl) UpdateEBook(ctx context.Context, id uuid.UUID, input UpdateEBookInput) (*models.EBook, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Title != nil {
		existing.Title = *input.Title
	}
	if input.Description != nil {
		existing.Description = *input.Description
	}
	if input.AuthorID != nil {
		existing.AuthorID = *input.AuthorID
	}
	if input.CoverURL != nil {
		existing.CoverURL = *input.CoverURL
	}
	existing.UpdatedAt = time.Now().UTC()
	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		return nil, err
	}
	_ = s.logService.RecordActivity(ctx, "ebook_updated", uuid.Nil)
	return updated, nil
}

func (s *EBookServiceImpl) DeleteEBook(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.logService.RecordActivity(ctx, "ebook_deleted", uuid.Nil)
	return nil
}
