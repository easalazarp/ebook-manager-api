package models

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrInvalidFormat se retorna cuando un formato de archivo no está entre los soportados.
// Permite a los llamadores distinguir este error con errors.As sin inspeccionar el texto.
type ErrInvalidFormat struct {
	Format string
}

func (e *ErrInvalidFormat) Error() string {
	return fmt.Sprintf("formato no soportado: %q (debe ser epub, pdf o mobi)", e.Format)
}

// IsErrInvalidFormat informa si err es de tipo ErrInvalidFormat.
func IsErrInvalidFormat(err error) bool {
	var target *ErrInvalidFormat
	return errors.As(err, &target)
}

// EBook representa la entidad principal del dominio: un libro digital.
type EBook struct {
	BaseModel
	AuthorID    uuid.UUID   `json:"author_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	CoverURL    string      `json:"cover_url,omitempty"`
	Files       []EBookFile `json:"files,omitempty"`
	Categories  []Category  `json:"categories,omitempty"`
}

// GetTitle devuelve el título del e-book.
func (e *EBook) GetTitle() string { return e.Title }

// GetCoverURL devuelve la URL de la portada; cadena vacía si no tiene portada asignada.
func (e *EBook) GetCoverURL() string { return e.CoverURL }

// GetFiles devuelve los archivos asociados al e-book.
func (e *EBook) GetFiles() []EBookFile { return e.Files }

// EBookFile representa un archivo de un e-book en un formato específico.
type EBookFile struct {
	ID      uuid.UUID `json:"id"`
	EBookID uuid.UUID `json:"ebook_id"`
	Format  string    `json:"format"` // "epub" | "pdf" | "mobi"
	FileURL string    `json:"file_url"`
}

// Format constants
const (
	FormatEPUB = "epub"
	FormatPDF  = "pdf"
	FormatMOBI = "mobi"
)

// ValidateFormat retorna *ErrInvalidFormat si f no es un formato soportado.
func ValidateFormat(f string) error {
	switch f {
	case FormatEPUB, FormatPDF, FormatMOBI:
		return nil
	default:
		return &ErrInvalidFormat{Format: f}
	}
}
