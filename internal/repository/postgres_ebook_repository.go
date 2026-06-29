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

var _ IBookRepository = (*PostgresEBookRepository)(nil)

type PostgresEBookRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresEBookRepository(pool *pgxpool.Pool) *PostgresEBookRepository {
	return &PostgresEBookRepository{pool: pool}
}

func (r *PostgresEBookRepository) Create(ctx context.Context, ebook *models.EBook) (*models.EBook, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres_ebook_repository.Create: begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	const insertEBook = `
		INSERT INTO ebooks (id, author_id, title, description, cover_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, author_id, title, description, cover_url, created_at, updated_at`

	row := tx.QueryRow(ctx, insertEBook,
		ebook.ID, ebook.AuthorID, ebook.Title, ebook.Description,
		ebook.CoverURL, ebook.CreatedAt, ebook.UpdatedAt,
	)
	created := &models.EBook{}
	if err = row.Scan(&created.ID, &created.AuthorID, &created.Title, &created.Description,
		&created.CoverURL, &created.CreatedAt, &created.UpdatedAt); err != nil {
		return nil, fmt.Errorf("postgres_ebook_repository.Create: insert ebook: %w", err)
	}

	const insertFile = `
		INSERT INTO ebook_files (id, ebook_id, format, file_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id, ebook_id, format, file_url`

	created.Files = make([]models.EBookFile, 0, len(ebook.Files))
	for _, f := range ebook.Files {
		fileRow := tx.QueryRow(ctx, insertFile, f.ID, created.ID, f.Format, f.FileURL)
		var ef models.EBookFile
		if err = fileRow.Scan(&ef.ID, &ef.EBookID, &ef.Format, &ef.FileURL); err != nil {
			return nil, fmt.Errorf("postgres_ebook_repository.Create: insert file %q: %w", f.Format, err)
		}
		created.Files = append(created.Files, ef)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("postgres_ebook_repository.Create: commit: %w", err)
	}
	return created, nil
}

func (r *PostgresEBookRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.EBook, error) {
	const query = `
		SELECT e.id, e.author_id, e.title, e.description, e.cover_url, e.created_at, e.updated_at,
		       f.id, f.ebook_id, f.format, f.file_url
		FROM ebooks e
		LEFT JOIN ebook_files f ON f.ebook_id = e.id
		WHERE e.id = $1
		ORDER BY f.format`

	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("postgres_ebook_repository.FindByID: query: %w", err)
	}
	defer rows.Close()

	var ebook *models.EBook
	for rows.Next() {
		var fileID, fileEBookID *uuid.UUID
		var format, fileURL *string
		if ebook == nil {
			ebook = &models.EBook{}
		}
		if err = rows.Scan(&ebook.ID, &ebook.AuthorID, &ebook.Title, &ebook.Description,
			&ebook.CoverURL, &ebook.CreatedAt, &ebook.UpdatedAt,
			&fileID, &fileEBookID, &format, &fileURL); err != nil {
			return nil, fmt.Errorf("postgres_ebook_repository.FindByID: scan: %w", err)
		}
		if fileID != nil {
			ebook.Files = append(ebook.Files, models.EBookFile{
				ID: *fileID, EBookID: *fileEBookID, Format: *format, FileURL: *fileURL,
			})
		}
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres_ebook_repository.FindByID: rows: %w", err)
	}
	if ebook == nil {
		return nil, fmt.Errorf("postgres_ebook_repository.FindByID: %w", ErrNotFound)
	}
	return ebook, nil
}

// List devuelve una página de e-books con el total de registros para el paginador.
// Usa dos queries: COUNT para el total y SELECT paginado para los datos.
func (r *PostgresEBookRepository) List(ctx context.Context, p PageParams) (Page[*models.EBook], error) {
	p = p.Clamp()

	// Query 1: contar el total de e-books distintos
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM ebooks`).Scan(&total); err != nil {
		return Page[*models.EBook]{}, fmt.Errorf("postgres_ebook_repository.List: count: %w", err)
	}

	// Query 2: e-books paginados con sus archivos (LEFT JOIN)
	// La subconsulta pagina los e-books y luego el JOIN trae los archivos,
	// evitando que LIMIT/OFFSET se apliquen sobre las filas duplicadas del JOIN.
	const query = `
		SELECT e.id, e.author_id, e.title, e.description, e.cover_url, e.created_at, e.updated_at,
		       f.id, f.ebook_id, f.format, f.file_url
		FROM (
			SELECT * FROM ebooks ORDER BY created_at DESC LIMIT $1 OFFSET $2
		) e
		LEFT JOIN ebook_files f ON f.ebook_id = e.id
		ORDER BY e.created_at DESC, f.format`

	rows, err := r.pool.Query(ctx, query, p.PageSize, p.Offset())
	if err != nil {
		return Page[*models.EBook]{}, fmt.Errorf("postgres_ebook_repository.List: query: %w", err)
	}
	defer rows.Close()

	index := make(map[uuid.UUID]*models.EBook)
	order := make([]uuid.UUID, 0, p.PageSize)

	for rows.Next() {
		var eb models.EBook
		var fileID, fileEBookID *uuid.UUID
		var format, fileURL *string
		if err = rows.Scan(&eb.ID, &eb.AuthorID, &eb.Title, &eb.Description,
			&eb.CoverURL, &eb.CreatedAt, &eb.UpdatedAt,
			&fileID, &fileEBookID, &format, &fileURL); err != nil {
			return Page[*models.EBook]{}, fmt.Errorf("postgres_ebook_repository.List: scan: %w", err)
		}
		existing, ok := index[eb.ID]
		if !ok {
			existing = &models.EBook{
				BaseModel: eb.BaseModel, AuthorID: eb.AuthorID,
				Title: eb.Title, Description: eb.Description, CoverURL: eb.CoverURL,
				Files: []models.EBookFile{},
			}
			index[eb.ID] = existing
			order = append(order, eb.ID)
		}
		if fileID != nil {
			existing.Files = append(existing.Files, models.EBookFile{
				ID: *fileID, EBookID: *fileEBookID, Format: *format, FileURL: *fileURL,
			})
		}
	}
	if err = rows.Err(); err != nil {
		return Page[*models.EBook]{}, fmt.Errorf("postgres_ebook_repository.List: rows: %w", err)
	}

	items := make([]*models.EBook, 0, len(order))
	for _, id := range order {
		items = append(items, index[id])
	}
	return NewPage(items, total, p.Page, p.PageSize), nil
}

func (r *PostgresEBookRepository) Update(ctx context.Context, ebook *models.EBook) (*models.EBook, error) {
	const query = `
		UPDATE ebooks SET title=$2, description=$3, author_id=$4, cover_url=$5, updated_at=now()
		WHERE id=$1
		RETURNING id, author_id, title, description, cover_url, created_at, updated_at`

	row := r.pool.QueryRow(ctx, query,
		ebook.ID, ebook.Title, ebook.Description, ebook.AuthorID, ebook.CoverURL)

	updated := &models.EBook{}
	err := row.Scan(&updated.ID, &updated.AuthorID, &updated.Title, &updated.Description,
		&updated.CoverURL, &updated.CreatedAt, &updated.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres_ebook_repository.Update: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("postgres_ebook_repository.Update: %w", err)
	}
	return updated, nil
}

func (r *PostgresEBookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM ebooks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres_ebook_repository.Delete: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("postgres_ebook_repository.Delete: %w", ErrNotFound)
	}
	return nil
}
