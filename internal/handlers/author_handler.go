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
	r.Get("/{id}", h.GetAuthor)
	r.Put("/{id}", h.UpdateAuthor)
	r.Delete("/{id}", h.DeleteAuthor)
}

// CreateAuthor godoc
//
//	@Summary		Crear autor
//	@Description	Crea un nuevo autor. Requiere autenticación JWT.
//	@Tags			autores
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		service.CreateAuthorInput	true	"Datos del autor"
//	@Success		201		{object}	models.Author
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/authors [post]
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

// ListAuthors godoc
//
//	@Summary		Listar autores
//	@Description	Devuelve la lista de todos los autores. Requiere autenticación JWT.
//	@Tags			autores
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		models.Author
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/authors [get]
func (h *AuthorHandler) ListAuthors(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.ListAuthors(r.Context(), repository.PageParams{Page: 1, PageSize: 200})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListAuthors: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, page.Items)
}

// GetAuthor godoc
//
//	@Summary		Obtener autor por ID
//	@Description	Devuelve un autor por su UUID. Requiere autenticación JWT.
//	@Tags			autores
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"UUID del autor"	format(uuid)
//	@Success		200	{object}	models.Author
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/authors/{id} [get]
func (h *AuthorHandler) GetAuthor(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UUID inválido")
		return
	}
	author, err := h.svc.GetAuthor(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "GetAuthor: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, author)
}

// UpdateAuthor godoc
//
//	@Summary		Actualizar autor
//	@Description	Actualiza los datos de un autor existente. Requiere autenticación JWT.
//	@Tags			autores
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"UUID del autor"	format(uuid)
//	@Param			body	body		service.UpdateAuthorInput	true	"Campos a actualizar"
//	@Success		200		{object}	models.Author
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/authors/{id} [put]
func (h *AuthorHandler) UpdateAuthor(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UUID inválido")
		return
	}
	var input service.UpdateAuthorInput
	if err = json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido: "+err.Error())
		return
	}
	author, err := h.svc.UpdateAuthor(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "UpdateAuthor: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, author)
}

// DeleteAuthor godoc
//
//	@Summary		Eliminar autor
//	@Description	Elimina un autor por su UUID. Requiere autenticación JWT.
//	@Tags			autores
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"UUID del autor"	format(uuid)
//	@Success		204	"Sin contenido"
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/authors/{id} [delete]
func (h *AuthorHandler) DeleteAuthor(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UUID inválido")
		return
	}
	if err = h.svc.DeleteAuthor(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "DeleteAuthor: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetAuthorByBookID godoc
//
//	@Summary		Obtener autor por ID de e-book
//	@Description	Devuelve el autor asociado a un e-book específico. Requiere autenticación JWT.
//	@Tags			autores
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"UUID del e-book"	format(uuid)
//	@Success		200	{object}	models.Author
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/ebooks/{id}/author [get]
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
