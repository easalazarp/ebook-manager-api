package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/ebook-management-backend/internal/models"
)

// Compile-time assertion: PostgresAuthorRepository must implement IAuthorRepository.
var _ IAuthorRepository = (*PostgresAuthorRepository)(nil)

// PostgresAuthorRepository implementa IAuthorRepository usando pgxpool.Pool.
type PostgresAuthorRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAuthorRepository crea una nueva instancia de PostgresAuthorRepository.
func NewPostgresAuthorRepository(pool *pgxpool.Pool) *PostgresAuthorRepository {
	return &PostgresAuthorRepository{pool: pool}
}

// Create inserta un nuevo autor en la base de datos.
func (r *PostgresAuthorRepository) Create(ctx context.Context, author *models.Author) (*models.Author, error) {
	const query = `
		INSERT INTO authors (id, name, bio, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, bio, created_at, updated_at`

	row := r.pool.QueryRow(ctx, query,
		author.ID, author.Name, author.Bio, author.CreatedAt, author.UpdatedAt,
	)
	created := &models.Author{}
	if err := row.Scan(&created.ID, &created.Name, &created.Bio, &created.CreatedAt, &created.UpdatedAt); err != nil {
		return nil, fmt.Errorf("postgres_author_repository.Create: %w", err)
	}
	return created, nil
}

// ListAll devuelve todos los autores ordenados por nombre ascendente.
func (r *PostgresAuthorRepository) ListAll(ctx context.Context) ([]*models.Author, error) {
	const query = `
		SELECT id, name, bio, created_at, updated_at
		FROM authors
		ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres_author_repository.ListAll: query: %w", err)
	}
	defer rows.Close()

	authors := make([]*models.Author, 0)
	for rows.Next() {
		a := &models.Author{}
		if err = rows.Scan(
			&a.ID,
			&a.Name,
			&a.Bio,
			&a.CreatedAt,
			&a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres_author_repository.ListAll: scan: %w", err)
		}
		authors = append(authors, a)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres_author_repository.ListAll: rows: %w", err)
	}

	return authors, nil
}

// GetByBookID recupera el autor asociado a un e-book dado su UUID.
// Retorna ErrNotFound si el e-book no existe o no tiene autor asignado.
func (r *PostgresAuthorRepository) GetByBookID(ctx context.Context, bookID uuid.UUID) (*models.Author, error) {
	const query = `
		SELECT a.id, a.name, a.bio, a.created_at, a.updated_at
		FROM authors a
		JOIN ebooks e ON e.author_id = a.id
		WHERE e.id = $1`

	row := r.pool.QueryRow(ctx, query, bookID)

	a := &models.Author{}
	err := row.Scan(
		&a.ID,
		&a.Name,
		&a.Bio,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres_author_repository.GetByBookID: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("postgres_author_repository.GetByBookID: %w", err)
	}

	return a, nil
}
