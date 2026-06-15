package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
)

// Compile-time assertion: EBookServiceImpl debe implementar IBookService.
// Si falta algún método la compilación falla, evitando errores en tiempo de ejecución.
var _ IBookService = (*EBookServiceImpl)(nil)

// EBookServiceImpl implementa IBookService.
// Coordina la validación de dominio, generación de IDs y registro de auditoría.
type EBookServiceImpl struct {
	repo       repository.IBookRepository
	logService ILogService
}

// NewEBookServiceImpl construye el servicio con sus dependencias inyectadas.
// Recibe interfaces, no implementaciones concretas, para facilitar testing con mocks.
func NewEBookServiceImpl(repo repository.IBookRepository, logService ILogService) *EBookServiceImpl {
	return &EBookServiceImpl{repo: repo, logService: logService}
}

// CreateEBook valida, construye y persiste un nuevo e-book con sus archivos.
//
// Flujo:
//  1. Valida el formato de cada archivo (epub/pdf/mobi) antes de persistir nada.
//     Usar models.IsErrInvalidFormat en el handler permite distinguir este error de un 500.
//  2. Genera los UUIDs y timestamps en el servicio (no en la BD), garantizando que
//     el objeto retornado tenga exactamente los mismos valores que se persistieron.
//  3. repo.Create usa una transacción SQL: si la inserción de algún archivo falla,
//     se hace rollback completo (el ebook tampoco queda guardado).
//  4. El log se registra con uuid.Nil como userID porque la capa de servicio no tiene
//     acceso directo al contexto de autenticación; en un sistema más complejo se
//     propagaría el userID desde el handler via context.
func (s *EBookServiceImpl) CreateEBook(ctx context.Context, input CreateEBookInput) (*models.EBook, error) {
	// Paso 1: validar todos los formatos antes de construir el objeto.
	// Fallar temprano evita trabajo innecesario y da mensajes de error precisos.
	for _, f := range input.Files {
		if err := models.ValidateFormat(f.Format); err != nil {
			return nil, err
		}
	}

	// Paso 2: construir el objeto en memoria con IDs y timestamps generados aquí.
	// uuid.New() genera un UUID v4 aleatorio, garantizando unicidad sin secuencias de BD.
	// time.Now().UTC() asegura zona horaria consistente independiente del servidor.
	now := time.Now().UTC()
	ebook := &models.EBook{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		AuthorID:    input.AuthorID,
		Title:       input.Title,
		Description: input.Description,
		Files:       make([]models.EBookFile, 0, len(input.Files)),
	}
	for _, f := range input.Files {
		ebook.Files = append(ebook.Files, models.EBookFile{
			ID:      uuid.New(),
			Format:  f.Format,
			FileURL: f.FileURL,
		})
	}

	// Paso 3: persistir. El repositorio maneja la transacción internamente.
	created, err := s.repo.Create(ctx, ebook)
	if err != nil {
		return nil, err
	}

	// Paso 4: registrar auditoría. El error se ignora intencionalmente:
	// un fallo de log no debe revertir una operación de negocio exitosa.
	_ = s.logService.RecordActivity(ctx, "ebook_created", uuid.Nil)
	return created, nil
}

// GetEBook recupera un e-book por su UUID incluyendo sus archivos.
// Retorna repository.ErrNotFound si no existe (detectado con errors.Is en el handler).
func (s *EBookServiceImpl) GetEBook(ctx context.Context, id uuid.UUID) (*models.EBook, error) {
	return s.repo.FindByID(ctx, id)
}

// ListEBooks retorna todos los e-books con sus archivos asociados.
// El repositorio garantiza que nunca retorna nil (como mínimo slice vacío).
func (s *EBookServiceImpl) ListEBooks(ctx context.Context) ([]*models.EBook, error) {
	return s.repo.List(ctx)
}

// UpdateEBook aplica una actualización parcial sobre un e-book existente.
//
// Por qué punteros en UpdateEBookInput:
//   - Un campo nil significa "no cambiar", no "poner vacío".
//   - Esto evita que enviar solo {"title":"X"} borre accidentalmente la descripción.
//   - Si el e-book no existe, repo.FindByID retorna ErrNotFound y el handler responde 404.
func (s *EBookServiceImpl) UpdateEBook(ctx context.Context, id uuid.UUID, input UpdateEBookInput) (*models.EBook, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Solo se modifica el campo si el puntero no es nil.
	if input.Title != nil {
		existing.Title = *input.Title
	}
	if input.Description != nil {
		existing.Description = *input.Description
	}
	existing.UpdatedAt = time.Now().UTC()

	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		return nil, err
	}
	_ = s.logService.RecordActivity(ctx, "ebook_updated", uuid.Nil)
	return updated, nil
}

// DeleteEBook elimina un e-book verificando primero su existencia.
// Los archivos asociados se eliminan automáticamente por la restricción CASCADE en BD.
func (s *EBookServiceImpl) DeleteEBook(ctx context.Context, id uuid.UUID) error {
	// FindByID asegura que retornamos ErrNotFound antes de intentar el DELETE.
	// Sin este check, un DELETE sobre un ID inexistente no daría error con pgx.
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.logService.RecordActivity(ctx, "ebook_deleted", uuid.Nil)
	return nil
}
