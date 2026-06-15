package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/middleware"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
	"github.com/yourorg/ebook-management-backend/internal/service"
)

// WebHandler maneja las vistas MVC públicas y la descarga protegida.
type WebHandler struct {
	svc       service.IBookService
	logSvc    service.ILogService
	authorSvc service.IAuthorService
}

func NewWebHandler(svc service.IBookService, logSvc service.ILogService, authorSvc service.IAuthorService) *WebHandler {
	return &WebHandler{svc: svc, logSvc: logSvc, authorSvc: authorSvc}
}

func isAuthenticated(r *http.Request) bool {
	_, err := r.Cookie("token")
	return err == nil
}

func (h *WebHandler) ListAuthorsWeb(w http.ResponseWriter, r *http.Request) {
	authors, err := h.authorSvc.ListAuthors(r.Context())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListAuthorsWeb: %v\n", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	if authors == nil {
		authors = []*models.Author{}
	}
	render(w, "authors.html", struct {
		Authors []*models.Author
		UserID  bool
	}{authors, isAuthenticated(r)})
}

func (h *WebHandler) ListCatalog(w http.ResponseWriter, r *http.Request) {
	ebooks, err := h.svc.ListEBooks(r.Context())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListCatalog: %v\n", err)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	if ebooks == nil {
		ebooks = []*models.EBook{}
	}
	render(w, "catalog.html", struct {
		EBooks []*models.EBook
		UserID bool
	}{ebooks, isAuthenticated(r)})
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
	render(w, "detail.html", struct {
		EBook  *models.EBook
		UserID bool
	}{ebook, isAuthenticated(r)})
}

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
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	var fileURL string
	for _, f := range ebook.Files {
		if f.Format == format {
			fileURL = f.FileURL
			break
		}
	}
	if fileURL == "" {
		http.Error(w, "formato no disponible", http.StatusNotFound)
		return
	}

	if userID, ok := middleware.UserIDFromContext(r.Context()); ok {
		_ = h.logSvc.RecordActivity(r.Context(), "ebook_downloaded", userID)
	}

	http.Redirect(w, r, fileURL, http.StatusFound)
}
