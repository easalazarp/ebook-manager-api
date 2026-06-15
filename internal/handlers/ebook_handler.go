package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
	"github.com/yourorg/ebook-management-backend/internal/service"
)

// EBookHandler maneja las rutas HTTP para e-books.
type EBookHandler struct {
	svc service.IBookService
}

func NewEBookHandler(svc service.IBookService) *EBookHandler {
	return &EBookHandler{svc: svc}
}

// Routes registra las rutas del handler en el router Chi proporcionado.
func (h *EBookHandler) Routes(r chi.Router) {
	r.Post("/", h.CreateEBook)
	r.Get("/", h.ListEBooks)
	r.Get("/{id}", h.GetEBook)
	r.Put("/{id}", h.UpdateEBook)
	r.Delete("/{id}", h.DeleteEBook)
}

func (h *EBookHandler) CreateEBook(w http.ResponseWriter, r *http.Request) {
	var input service.CreateEBookInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido: "+err.Error())
		return
	}

	ebook, err := h.svc.CreateEBook(r.Context(), input)
	if err != nil {
		if models.IsErrInvalidFormat(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ebook)
}

func (h *EBookHandler) GetEBook(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UUID inválido")
		return
	}

	ebook, err := h.svc.GetEBook(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "GetEBook: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, ebook)
}

func (h *EBookHandler) ListEBooks(w http.ResponseWriter, r *http.Request) {
	ebooks, err := h.svc.ListEBooks(r.Context())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListEBooks: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if ebooks == nil {
		ebooks = []*models.EBook{}
	}
	writeJSON(w, http.StatusOK, ebooks)
}

func (h *EBookHandler) UpdateEBook(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UUID inválido")
		return
	}

	var input service.UpdateEBookInput
	if err = json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido: "+err.Error())
		return
	}

	ebook, err := h.svc.UpdateEBook(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "UpdateEBook: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, ebook)
}

func (h *EBookHandler) DeleteEBook(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UUID inválido")
		return
	}

	if err = h.svc.DeleteEBook(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "DeleteEBook: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
