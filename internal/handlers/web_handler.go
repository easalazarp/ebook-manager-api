package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/middleware"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
	"github.com/yourorg/ebook-management-backend/internal/service"
	"github.com/yourorg/ebook-management-backend/internal/storage"
)

// Tiempos de expiración para las signed URLs de descarga.
// Las portadas usan URL pública directa (no expiran).
const downloadSignedURLExpiry = 300 // 5 minutos

// WebHandler maneja las vistas MVC públicas y la descarga protegida.
type WebHandler struct {
	svc           service.IBookService
	logSvc        service.ILogService
	authorSvc     service.IAuthorService
	storageClient *storage.Client
}

// NewWebHandler construye el WebHandler.
func NewWebHandler(svc service.IBookService, logSvc service.ILogService, authorSvc service.IAuthorService, sc *storage.Client) *WebHandler {
	return &WebHandler{svc: svc, logSvc: logSvc, authorSvc: authorSvc, storageClient: sc}
}

func isAuthenticated(r *http.Request) bool {
	_, err := r.Cookie("token")
	return err == nil
}

// ── Vistas públicas ───────────────────────────────────────────────────────

func (h *WebHandler) ListAuthorsWeb(w http.ResponseWriter, r *http.Request) {
	page, err := h.authorSvc.ListAuthors(r.Context(), repository.PageParams{Page: 1, PageSize: 50})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListAuthorsWeb: %v\n", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	render(w, "authors.html", struct {
		Authors []*models.Author
		UserID  bool
	}{page.Items, isAuthenticated(r)})
}

func (h *WebHandler) ListCatalog(w http.ResponseWriter, r *http.Request) {
	pg, err := h.svc.ListEBooks(r.Context(), repository.PageParams{Page: 1, PageSize: 50})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListCatalog: %v\n", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	h.resolveCovers(pg.Items)
	render(w, "catalog.html", struct {
		EBooks []*models.EBook
		UserID bool
	}{pg.Items, isAuthenticated(r)})
}

func (h *WebHandler) ShowEBook(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ebook, err := h.svc.GetEBook(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(os.Stderr, "ShowEBook: %v\n", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	// Resolver URL pública de portada para este libro.
	h.resolveCovers([]*models.EBook{ebook})

	render(w, "detail.html", struct {
		EBook  *models.EBook
		UserID bool
	}{ebook, isAuthenticated(r)})
}

// ── Descarga protegida ────────────────────────────────────────────────────

// DownloadEBook sirve la descarga redirigiendo a la URL pública del archivo en Storage.
// Con bucket público, la URL es permanente y no necesita signed token.
// El middleware RequireAuth en la ruta garantiza que solo usuarios autenticados lleguen aquí.
func (h *WebHandler) DownloadEBook(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	format := r.URL.Query().Get("format")

	ebook, err := h.svc.GetEBook(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(os.Stderr, "DownloadEBook: GetEBook: %v\n", err)
		http.Error(w, "No se pudo procesar la descarga.", http.StatusInternalServerError)
		return
	}

	// Buscar el archivo del formato solicitado
	var filePath string
	for _, f := range ebook.Files {
		if f.Format == format {
			filePath = f.FileURL
			break
		}
	}
	if filePath == "" {
		http.Error(w, "Formato no disponible para este e-book.", http.StatusNotFound)
		return
	}

	// Auditoría (error ignorado — resiliencia de logs)
	if userID, ok := middleware.UserIDFromContext(r.Context()); ok {
		_ = h.logSvc.RecordActivity(r.Context(), fmt.Sprintf("ebook_downloaded:%s", id), userID)
	}

	if h.storageClient == nil {
		http.Error(w, "Servicio de almacenamiento no disponible.", http.StatusServiceUnavailable)
		return
	}

	// Construir URL pública directa (bucket público, sin signed token).
	// El control de acceso lo ejerce RequireAuth en el router — solo usuarios
	// con JWT válido llegan hasta aquí.
	downloadURL := h.storageClient.PublicURL(filePath)
	http.Redirect(w, r, downloadURL, http.StatusFound)
}

// ── Helpers privados ──────────────────────────────────────────────────────

// resolveCovers construye la URL pública de portada para cada libro.
// El bucket es público, así que la URL es permanente y no necesita signed token.
// Formato: https://<project>.supabase.co/storage/v1/object/public/<bucket>/<path>
//
// Si la BD ya guarda una URL completa (datos migrados) la usa tal cual.
// Si guarda un path relativo (datos nuevos) construye la URL pública.
func (h *WebHandler) resolveCovers(ebooks []*models.EBook) {
	if h.storageClient == nil {
		return
	}
	for _, eb := range ebooks {
		if eb.CoverURL == "" {
			continue
		}
		eb.CoverURL = h.storageClient.PublicURL(eb.CoverURL)
	}
}

// normalizeStoragePath delega al storage client.
func normalizeStoragePath(raw string) string {
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		return raw
	}
	for _, marker := range []string{
		"/object/public/",
		"/object/sign/",
		"/object/authenticated/",
		"/object/",
	} {
		idx := strings.Index(raw, marker)
		if idx == -1 {
			continue
		}
		rest := raw[idx+len(marker):]
		if q := strings.Index(rest, "?"); q != -1 {
			rest = rest[:q]
		}
		slash := strings.Index(rest, "/")
		if slash == -1 {
			return ""
		}
		return rest[slash+1:]
	}
	return raw
}
