package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/ebook-management-backend/internal/models"
)

var _ ICategoryRepository = (*PostgresCategoryRepository)(nil)

type PostgresCategoryRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCategoryRepository(pool *pgxpool.Pool) *PostgresCategoryRepository {
	return &PostgresCategoryRepository{pool: pool}
}

func (r *PostgresCategoryRepository) Create(ctx context.Context, category *models.Category) (*models.Category, error) {
	const query = `
		INSERT INTO categories (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, slug, created_at, updated_at`

	row := r.pool.QueryRow(ctx, query,
		category.ID, category.Name, category.Slug, category.CreatedAt, category.UpdatedAt)
	created := &models.Category{}
	if err := row.Scan(&created.ID, &created.Name, &created.Slug, &created.CreatedAt, &created.UpdatedAt); err != nil {
		return nil, fmt.Errorf("postgres_category_repository.Create: %w", err)
	}
	return created, nil
}

func (r *PostgresCategoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	const query = `SELECT id, name, slug, created_at, updated_at FROM categories WHERE id = $1`
	c := &models.Category{}
	err := r.pool.QueryRow(ctx, query, id).Scan(&c.ID, &c.Name, &c.Slug, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres_category_repository.FindByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("postgres_category_repository.FindByID: %w", err)
	}
	return c, nil
}

func (r *PostgresCategoryRepository) ListAll(ctx context.Context, p PageParams) (Page[*models.Category], error) {
	p = p.Clamp()

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM categories`).Scan(&total); err != nil {
		return Page[*models.Category]{}, fmt.Errorf("postgres_category_repository.ListAll: count: %w", err)
	}

	const query = `
		SELECT id, name, slug, created_at, updated_at
		FROM categories ORDER BY name ASC LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, p.PageSize, p.Offset())
	if err != nil {
		return Page[*models.Category]{}, fmt.Errorf("postgres_category_repository.ListAll: query: %w", err)
	}
	defer rows.Close()

	categories := make([]*models.Category, 0, p.PageSize)
	for rows.Next() {
		c := &models.Category{}
		if err = rows.Scan(&c.ID, &c.Name, &c.Slug, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return Page[*models.Category]{}, fmt.Errorf("postgres_category_repository.ListAll: scan: %w", err)
		}
		categories = append(categories, c)
	}
	if err = rows.Err(); err != nil {
		return Page[*models.Category]{}, fmt.Errorf("postgres_category_repository.ListAll: rows: %w", err)
	}
	return NewPage(categories, total, p.Page, p.PageSize), nil
}

func (r *PostgresCategoryRepository) Update(ctx context.Context, category *models.Category) (*models.Category, error) {
	const query = `
		UPDATE categories SET name=$2, slug=$3, updated_at=$4 WHERE id=$1
		RETURNING id, name, slug, created_at, updated_at`

	row := r.pool.QueryRow(ctx, query, category.ID, category.Name, category.Slug, time.Now().UTC())
	updated := &models.Category{}
	err := row.Scan(&updated.ID, &updated.Name, &updated.Slug, &updated.CreatedAt, &updated.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres_category_repository.Update: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("postgres_category_repository.Update: %w", err)
	}
	return updated, nil
}

func (r *PostgresCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres_category_repository.Delete: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("postgres_category_repository.Delete: %w", ErrNotFound)
	}
	return nil
}
