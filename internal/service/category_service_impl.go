package service

import (
	"context"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
)

var _ ICategoryService = (*CategoryServiceImpl)(nil)

type CategoryServiceImpl struct {
	repo repository.ICategoryRepository
}

func NewCategoryServiceImpl(repo repository.ICategoryRepository) *CategoryServiceImpl {
	return &CategoryServiceImpl{repo: repo}
}

func (s *CategoryServiceImpl) CreateCategory(ctx context.Context, input CreateCategoryInput) (*models.Category, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, &ErrValidation{Field: "name", Msg: "el nombre de la categoría es obligatorio"}
	}
	category := models.NewCategory(name, slugify(name))
	return s.repo.Create(ctx, category)
}

func (s *CategoryServiceImpl) GetCategory(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *CategoryServiceImpl) ListCategories(ctx context.Context, p repository.PageParams) (repository.Page[*models.Category], error) {
	return s.repo.ListAll(ctx, p)
}

func (s *CategoryServiceImpl) UpdateCategory(ctx context.Context, id uuid.UUID, input UpdateCategoryInput) (*models.Category, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, &ErrValidation{Field: "name", Msg: "el nombre de la categoría es obligatorio"}
		}
		existing.Name = name
		existing.Slug = slugify(name)
	}
	return s.repo.Update(ctx, existing)
}

func (s *CategoryServiceImpl) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// slugify convierte un nombre en slug URL-friendly: "Ciencia Ficción" → "ciencia-ficcion"
func slugify(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	result = strings.ToLower(result)
	var b strings.Builder
	prevDash := false
	for _, r := range result {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// ErrValidation es un error de validación de dominio con campo y mensaje.
type ErrValidation struct {
	Field string
	Msg   string
}

func (e *ErrValidation) Error() string { return e.Msg }
