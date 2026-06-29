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

var _ IAuthorRepository = (*PostgresAuthorRepository)(nil)

type PostgresAuthorRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAuthorRepository(pool *pgxpool.Pool) *PostgresAuthorRepository {
	return &PostgresAuthorRepository{pool: pool}
}

func (r *PostgresAuthorRepository) Create(ctx context.Context, author *models.Author) (*models.Author, error) {
	const query = `
		INSERT INTO authors (id, name, bio, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, bio, created_at, updated_at`

	row := r.pool.QueryRow(ctx, query, author.ID, author.Name, author.Bio, author.CreatedAt, author.UpdatedAt)
	created := &models.Author{}
	if err := row.Scan(&created.ID, &created.Name, &created.Bio, &created.CreatedAt, &created.UpdatedAt); err != nil {
		return nil, fmt.Errorf("postgres_author_repository.Create: %w", err)
	}
	return created, nil
}

func (r *PostgresAuthorRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Author, error) {
	const query = `SELECT id, name, bio, created_at, updated_at FROM authors WHERE id = $1`
	a := &models.Author{}
	err := r.pool.QueryRow(ctx, query, id).Scan(&a.ID, &a.Name, &a.Bio, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres_author_repository.FindByID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("postgres_author_repository.FindByID: %w", err)
	}
	return a, nil
}

func (r *PostgresAuthorRepository) ListAll(ctx context.Context, p PageParams) (Page[*models.Author], error) {
	p = p.Clamp()

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM authors`).Scan(&total); err != nil {
		return Page[*models.Author]{}, fmt.Errorf("postgres_author_repository.ListAll: count: %w", err)
	}

	const query = `
		SELECT id, name, bio, created_at, updated_at
		FROM authors ORDER BY name ASC LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, p.PageSize, p.Offset())
	if err != nil {
		return Page[*models.Author]{}, fmt.Errorf("postgres_author_repository.ListAll: query: %w", err)
	}
	defer rows.Close()

	authors := make([]*models.Author, 0, p.PageSize)
	for rows.Next() {
		a := &models.Author{}
		if err = rows.Scan(&a.ID, &a.Name, &a.Bio, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return Page[*models.Author]{}, fmt.Errorf("postgres_author_repository.ListAll: scan: %w", err)
		}
		authors = append(authors, a)
	}
	if err = rows.Err(); err != nil {
		return Page[*models.Author]{}, fmt.Errorf("postgres_author_repository.ListAll: rows: %w", err)
	}
	return NewPage(authors, total, p.Page, p.PageSize), nil
}

func (r *PostgresAuthorRepository) Update(ctx context.Context, author *models.Author) (*models.Author, error) {
	const query = `
		UPDATE authors SET name=$2, bio=$3, updated_at=$4 WHERE id=$1
		RETURNING id, name, bio, created_at, updated_at`

	row := r.pool.QueryRow(ctx, query, author.ID, author.Name, author.Bio, time.Now().UTC())
	updated := &models.Author{}
	err := row.Scan(&updated.ID, &updated.Name, &updated.Bio, &updated.CreatedAt, &updated.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres_author_repository.Update: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("postgres_author_repository.Update: %w", err)
	}
	return updated, nil
}

func (r *PostgresAuthorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM authors WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres_author_repository.Delete: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("postgres_author_repository.Delete: %w", ErrNotFound)
	}
	return nil
}

func (r *PostgresAuthorRepository) GetByBookID(ctx context.Context, bookID uuid.UUID) (*models.Author, error) {
	const query = `
		SELECT a.id, a.name, a.bio, a.created_at, a.updated_at
		FROM authors a JOIN ebooks e ON e.author_id = a.id WHERE e.id = $1`

	a := &models.Author{}
	err := r.pool.QueryRow(ctx, query, bookID).Scan(&a.ID, &a.Name, &a.Bio, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres_author_repository.GetByBookID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("postgres_author_repository.GetByBookID: %w", err)
	}
	return a, nil
}
