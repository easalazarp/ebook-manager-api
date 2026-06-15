package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/ebook-management-backend/internal/models"
)

// Compile-time assertion: PostgresLogRepository must implement ILogRepository.
var _ ILogRepository = (*PostgresLogRepository)(nil)

// PostgresLogRepository implementa ILogRepository usando pgxpool.Pool.
type PostgresLogRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresLogRepository crea una nueva instancia de PostgresLogRepository.
func NewPostgresLogRepository(pool *pgxpool.Pool) *PostgresLogRepository {
	return &PostgresLogRepository{pool: pool}
}

// SaveLog inserta un nuevo registro de auditoría en la tabla logs.
func (r *PostgresLogRepository) SaveLog(ctx context.Context, log *models.Log) error {
	const query = `
		INSERT INTO logs (id, activity, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	now := time.Now().UTC()
	if log.CreatedAt.IsZero() {
		log.CreatedAt = now
	}
	if log.UpdatedAt.IsZero() {
		log.UpdatedAt = now
	}

	_, err := r.pool.Exec(ctx, query,
		log.ID,
		log.Activity,
		log.UserID,
		log.CreatedAt,
		log.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres_log_repository.SaveLog: %w", err)
	}

	return nil
}

// GetHistory devuelve todos los registros de auditoría ordenados por created_at descendente.
func (r *PostgresLogRepository) GetHistory(ctx context.Context) ([]models.Log, error) {
	const query = `
		SELECT id, activity, user_id, created_at, updated_at
		FROM logs
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres_log_repository.GetHistory: query: %w", err)
	}
	defer rows.Close()

	logs := make([]models.Log, 0)
	for rows.Next() {
		var l models.Log
		if err = rows.Scan(
			&l.ID,
			&l.Activity,
			&l.UserID,
			&l.CreatedAt,
			&l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres_log_repository.GetHistory: scan: %w", err)
		}
		logs = append(logs, l)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres_log_repository.GetHistory: rows: %w", err)
	}

	return logs, nil
}
