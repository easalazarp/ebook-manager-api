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

// Compile-time assertion: PostgresEBookRepository must implement IBookRepository.
var _ IBookRepository = (*PostgresEBookRepository)(nil)

// PostgresEBookRepository implementa IBookRepository usando pgxpool.Pool.
type PostgresEBookRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresEBookRepository crea una nueva instancia de PostgresEBookRepository.
func NewPostgresEBookRepository(pool *pgxpool.Pool) *PostgresEBookRepository {
	return &PostgresEBookRepository{pool: pool}
}

// Create inserta un e-book y sus archivos en la base de datos.
//
// Se usa una TRANSACCIÓN para garantizar atomicidad:
//   - Si la inserción de algún EBookFile falla, se hace ROLLBACK del ebook también.
//   - El defer con la variable de error capturada asegura que el rollback ocurre
//     solo cuando err != nil (si Commit fue exitoso, err será nil y no se rollback).
//
// Retorna el EBook completo con los IDs y timestamps confirmados por la BD via RETURNING.
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
		INSERT INTO ebooks (id, author_id, title, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, author_id, title, description, created_at, updated_at`

	row := tx.QueryRow(ctx, insertEBook,
		ebook.ID,
		ebook.AuthorID,
		ebook.Title,
		ebook.Description,
		ebook.CreatedAt,
		ebook.UpdatedAt,
	)

	created := &models.EBook{}
	if err = row.Scan(
		&created.ID,
		&created.AuthorID,
		&created.Title,
		&created.Description,
		&created.CreatedAt,
		&created.UpdatedAt,
	); err != nil {
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

// FindByID recupera un e-book por su UUID, incluyendo sus archivos.
//
// Estrategia LEFT JOIN en una sola consulta:
//   - Un e-book sin archivos retorna una fila con todos los campos de ebook_files en NULL.
//   - Por eso las columnas del archivo se escanean en punteros (*uuid.UUID, *string, etc.)
//     y solo se agregan al slice si fileID != nil.
//
// Retorna ErrNotFound cuando el e-book no existe (ebook == nil al final del loop).
func (r *PostgresEBookRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.EBook, error) {
	const query = `
		SELECT
			e.id, e.author_id, e.title, e.description, e.created_at, e.updated_at,
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
		var (
			fileID      *uuid.UUID
			fileEBookID *uuid.UUID
			format      *string
			fileURL     *string
		)
		if ebook == nil {
			ebook = &models.EBook{}
		}
		if err = rows.Scan(
			&ebook.ID,
			&ebook.AuthorID,
			&ebook.Title,
			&ebook.Description,
			&ebook.CreatedAt,
			&ebook.UpdatedAt,
			&fileID,
			&fileEBookID,
			&format,
			&fileURL,
		); err != nil {
			return nil, fmt.Errorf("postgres_ebook_repository.FindByID: scan: %w", err)
		}
		if fileID != nil {
			ebook.Files = append(ebook.Files, models.EBookFile{
				ID:      *fileID,
				EBookID: *fileEBookID,
				Format:  *format,
				FileURL: *fileURL,
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

// List devuelve todos los e-books con sus archivos asociados.
//
// El JOIN produce múltiples filas por e-book (una por archivo). Para reconstruir
// la estructura jerárquica en Go se usa:
//   - index map[uuid.UUID]*EBook: para encontrar el ebook padre en O(1) y agregar archivos.
//   - order []uuid.UUID: para preservar el orden original de la consulta (ORDER BY created_at DESC)
//     ya que los maps en Go no garantizan orden de iteración.
func (r *PostgresEBookRepository) List(ctx context.Context) ([]*models.EBook, error) {
	const query = `
		SELECT
			e.id, e.author_id, e.title, e.description, e.created_at, e.updated_at,
			f.id, f.ebook_id, f.format, f.file_url
		FROM ebooks e
		LEFT JOIN ebook_files f ON f.ebook_id = e.id
		ORDER BY e.created_at DESC, f.format`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres_ebook_repository.List: query: %w", err)
	}
	defer rows.Close()

	// index preserves insertion order; map groups files per ebook
	index := make(map[uuid.UUID]*models.EBook)
	order := make([]uuid.UUID, 0)

	for rows.Next() {
		var (
			eb          models.EBook
			fileID      *uuid.UUID
			fileEBookID *uuid.UUID
			format      *string
			fileURL     *string
		)
		if err = rows.Scan(
			&eb.ID,
			&eb.AuthorID,
			&eb.Title,
			&eb.Description,
			&eb.CreatedAt,
			&eb.UpdatedAt,
			&fileID,
			&fileEBookID,
			&format,
			&fileURL,
		); err != nil {
			return nil, fmt.Errorf("postgres_ebook_repository.List: scan: %w", err)
		}

		existing, ok := index[eb.ID]
		if !ok {
			existing = &models.EBook{
				BaseModel:   eb.BaseModel,
				AuthorID:    eb.AuthorID,
				Title:       eb.Title,
				Description: eb.Description,
				Files:       []models.EBookFile{},
			}
			index[eb.ID] = existing
			order = append(order, eb.ID)
		}

		if fileID != nil {
			existing.Files = append(existing.Files, models.EBookFile{
				ID:      *fileID,
				EBookID: *fileEBookID,
				Format:  *format,
				FileURL: *fileURL,
			})
		}
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres_ebook_repository.List: rows: %w", err)
	}

	result := make([]*models.EBook, 0, len(order))
	for _, id := range order {
		result = append(result, index[id])
	}
	return result, nil
}

// Update actualiza los campos modificables de un e-book existente.
// El trigger de la BD se encarga de actualizar updated_at automáticamente,
// pero también lo enviamos explícitamente para obtener el valor correcto en RETURNING.
func (r *PostgresEBookRepository) Update(ctx context.Context, ebook *models.EBook) (*models.EBook, error) {
	const query = `
		UPDATE ebooks
		SET
			title       = $2,
			description = $3,
			author_id   = $4,
			updated_at  = now()
		WHERE id = $1
		RETURNING id, author_id, title, description, created_at, updated_at`

	row := r.pool.QueryRow(ctx, query,
		ebook.ID,
		ebook.Title,
		ebook.Description,
		ebook.AuthorID,
	)

	updated := &models.EBook{}
	err := row.Scan(
		&updated.ID,
		&updated.AuthorID,
		&updated.Title,
		&updated.Description,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("postgres_ebook_repository.Update: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("postgres_ebook_repository.Update: %w", err)
	}

	return updated, nil
}

// Delete elimina un e-book por su UUID.
// Las tablas ebook_files y ebook_categories se limpian automáticamente via CASCADE.
// Retorna ErrNotFound si el e-book no existe.
func (r *PostgresEBookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM ebooks WHERE id = $1`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("postgres_ebook_repository.Delete: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("postgres_ebook_repository.Delete: %w", ErrNotFound)
	}
	return nil
}
