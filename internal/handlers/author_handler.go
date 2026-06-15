package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/repository"
	"github.com/yourorg/ebook-management-backend/internal/service"
)

// AuthorHandler maneja las rutas HTTP para autores.
type AuthorHandler struct {
	svc service.IAuthorService
}

func NewAuthorHandler(svc service.IAuthorService) *AuthorHandler {
	return &AuthorHandler{svc: svc}
}

// Routes registra las rutas del handler en el router Chi proporcionado.
func (h *AuthorHandler) Routes(r chi.Router) {
	r.Post("/", h.CreateAuthor)
	r.Get("/", h.ListAuthors)
}

func (h *AuthorHandler) CreateAuthor(w http.ResponseWriter, r *http.Request) {
	var input service.CreateAuthorInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido: "+err.Error())
		return
	}
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "el campo name es requerido")
		return
	}
	author, err := h.svc.CreateAuthor(r.Context(), input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateAuthor: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, author)
}

func (h *AuthorHandler) ListAuthors(w http.ResponseWriter, r *http.Request) {
	authors, err := h.svc.ListAuthors(r.Context())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListAuthors: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, authors)
}

func (h *AuthorHandler) GetAuthorByBookID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UUID inválido")
		return
	}

	author, err := h.svc.GetAuthorByBookID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "GetAuthorByBookID: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, author)
}
