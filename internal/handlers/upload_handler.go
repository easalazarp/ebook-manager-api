package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/service"
	"github.com/yourorg/ebook-management-backend/internal/storage"
)

const (
	uploadMaxBookSize  = 50 << 20 // 50 MB
	uploadMaxCoverSize = 5 << 20  // 5 MB
)

// UploadHandler gestiona la subida de portadas y archivos de e-books vía API REST.
type UploadHandler struct {
	ebookSvc      service.IBookService
	storageClient *storage.Client
}

// NewUploadHandler crea un nuevo UploadHandler.
func NewUploadHandler(ebookSvc service.IBookService, sc *storage.Client) *UploadHandler {
	return &UploadHandler{ebookSvc: ebookSvc, storageClient: sc}
}

// Routes registra las rutas del handler en el router Chi proporcionado.
func (h *UploadHandler) Routes(r chi.Router) {
	r.Post("/ebooks/{id}/cover", h.UploadCover)
	r.Post("/ebooks/{id}/file", h.UploadFile)
}

// ─── DTOs de respuesta ───────────────────────────────────────────────────────

// UploadCoverResponse es la respuesta tras subir una portada exitosamente.
type UploadCoverResponse struct {
	// Path relativo del archivo en el bucket de Storage.
	// Úsalo para construir la URL pública o una signed URL.
	// example: covers/550e8400-e29b-41d4-a716-446655440000.jpg
	CoverPath string `json:"cover_path" example:"covers/550e8400-e29b-41d4-a716-446655440000.jpg"`
	// URL pública de la portada (solo si el bucket es público).
	// example: https://xxxx.supabase.co/storage/v1/object/public/ebooks/covers/uuid.jpg
	CoverURL string `json:"cover_url" example:"https://xxxx.supabase.co/storage/v1/object/public/ebooks/covers/uuid.jpg"`
}

// UploadFileResponse es la respuesta tras subir un archivo de e-book exitosamente.
type UploadFileResponse struct {
	// Path relativo del archivo en el bucket de Storage.
	// example: pdf/550e8400-e29b-41d4-a716-446655440000.pdf
	FilePath string `json:"file_path" example:"pdf/550e8400-e29b-41d4-a716-446655440000.pdf"`
	// URL pública del archivo (solo si el bucket es público).
	// example: https://xxxx.supabase.co/storage/v1/object/public/ebooks/pdf/uuid.pdf
	FileURL string `json:"file_url" example:"https://xxxx.supabase.co/storage/v1/object/public/ebooks/pdf/uuid.pdf"`
	// Formato del archivo subido (epub, pdf o mobi).
	// example: pdf
	Format string `json:"format" example:"pdf"`
}

var coverMIMETypes = map[string]string{
	".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".png": "image/png", ".webp": "image/webp",
}

var bookMIMETypes = map[string]string{
	models.FormatPDF:  "application/pdf",
	models.FormatEPUB: "application/epub+zip",
	models.FormatMOBI: "application/x-mobipocket-ebook",
}

// ─── Handlers ───────────────────────────────────────────────────────────────

// UploadCover godoc
//
//	@Summary		Subir portada de e-book
//	@Description	Sube una imagen de portada para un e-book existente y actualiza su registro.
//	@Description	Formatos aceptados: JPG, JPEG, PNG, WebP. Tamaño máximo: 5 MB.
//	@Description	Requiere autenticación JWT.
//	@Tags			uploads
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string	true	"UUID del e-book"	format(uuid)
//	@Param			cover	formData	file	true	"Imagen de portada (JPG/PNG/WebP, máx 5 MB)"
//	@Success		200		{object}	UploadCoverResponse
//	@Failure		400		{object}	map[string]string	"UUID inválido, formato no soportado o archivo muy grande"
//	@Failure		401		{object}	map[string]string	"No autenticado"
//	@Failure		404		{object}	map[string]string	"E-book no encontrado"
//	@Failure		500		{object}	map[string]string	"Error interno o de storage"
//	@Router			/ebooks/{id}/cover [post]
func (h *UploadHandler) UploadCover(w http.ResponseWriter, r *http.Request) {
	ebookID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UUID inválido")
		return
	}

	// Verificar que el e-book existe antes de subir el archivo
	existing, err := h.ebookSvc.GetEBook(r.Context(), ebookID)
	if err != nil {
		writeError(w, http.StatusNotFound, "e-book no encontrado")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, uploadMaxCoverSize+4096)
	if err = r.ParseMultipartForm(uploadMaxCoverSize); err != nil {
		writeError(w, http.StatusBadRequest, "archivo demasiado grande (máx 5 MB)")
		return
	}

	file, header, err := r.FormFile("cover")
	if err != nil {
		writeError(w, http.StatusBadRequest, "campo 'cover' requerido")
		return
	}
	defer file.Close()

	if header.Size > uploadMaxCoverSize {
		writeError(w, http.StatusBadRequest, "la portada no puede superar los 5 MB")
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	ct, ok := coverMIMETypes[ext]
	if !ok {
		writeError(w, http.StatusBadRequest, "formato no soportado. Usa: jpg, jpeg, png o webp")
		return
	}

	coverPath, err := h.storageClient.Upload(
		r.Context(),
		fmt.Sprintf("covers/%s%s", uuid.New(), ext),
		ct,
		file,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "UploadCover: storage upload: %v\n", err)
		writeError(w, http.StatusInternalServerError, "no se pudo subir la portada")
		return
	}

	// Actualizar el e-book con la nueva portada
	_, err = h.ebookSvc.UpdateEBook(r.Context(), existing.ID, service.UpdateEBookInput{
		CoverURL: &coverPath,
	})
	if err != nil {
		// Intentar limpiar el archivo subido si la actualización falla
		_ = h.storageClient.Delete(r.Context(), coverPath)
		fmt.Fprintf(os.Stderr, "UploadCover: update ebook: %v\n", err)
		writeError(w, http.StatusInternalServerError, "no se pudo actualizar el e-book")
		return
	}

	writeJSON(w, http.StatusOK, UploadCoverResponse{
		CoverPath: coverPath,
		CoverURL:  h.storageClient.PublicURL(coverPath),
	})
}

// UploadFile godoc
//
//	@Summary		Subir archivo de e-book
//	@Description	Sube un archivo de e-book (PDF, EPUB o MOBI) y lo asocia al e-book indicado.
//	@Description	Tamaño máximo: 50 MB. Requiere autenticación JWT.
//	@Tags			uploads
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string	true	"UUID del e-book"							format(uuid)
//	@Param			file	formData	file	true	"Archivo del libro (PDF/EPUB/MOBI, máx 50 MB)"
//	@Param			format	formData	string	true	"Formato del archivo"						Enums(pdf, epub, mobi)
//	@Success		200		{object}	UploadFileResponse
//	@Failure		400		{object}	map[string]string	"UUID inválido, formato no soportado o archivo muy grande"
//	@Failure		401		{object}	map[string]string	"No autenticado"
//	@Failure		404		{object}	map[string]string	"E-book no encontrado"
//	@Failure		500		{object}	map[string]string	"Error interno o de storage"
//	@Router			/ebooks/{id}/file [post]
func (h *UploadHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	ebookID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UUID inválido")
		return
	}

	// Verificar que el e-book existe
	if _, err = h.ebookSvc.GetEBook(r.Context(), ebookID); err != nil {
		writeError(w, http.StatusNotFound, "e-book no encontrado")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, uploadMaxBookSize+4096)
	if err = r.ParseMultipartForm(uploadMaxBookSize); err != nil {
		writeError(w, http.StatusBadRequest, "archivo demasiado grande (máx 50 MB)")
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.FormValue("format")))
	if err = models.ValidateFormat(format); err != nil {
		writeError(w, http.StatusBadRequest, "formato no soportado. Usa: epub, pdf o mobi")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "campo 'file' requerido")
		return
	}
	defer file.Close()

	if header.Size > uploadMaxBookSize {
		writeError(w, http.StatusBadRequest, "el archivo no puede superar los 50 MB")
		return
	}

	ct := bookMIMETypes[format]
	if ct == "" {
		ct = "application/octet-stream"
	}

	filePath := fmt.Sprintf("%s/%s.%s", format, uuid.New(), format)
	fileURL, err := h.storageClient.Upload(r.Context(), filePath, ct, file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "UploadFile: storage upload: %v\n", err)
		writeError(w, http.StatusInternalServerError, "no se pudo subir el archivo")
		return
	}

	writeJSON(w, http.StatusOK, UploadFileResponse{
		FilePath: fileURL,
		FileURL:  h.storageClient.PublicURL(fileURL),
		Format:   format,
	})
}
