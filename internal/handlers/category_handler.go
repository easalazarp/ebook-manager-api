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

// CategoryHandler maneja las rutas HTTP para categorías.
type CategoryHandler struct {
	svc service.ICategoryService
}

// NewCategoryHandler crea un nuevo CategoryHandler.
func NewCategoryHandler(svc service.ICategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

// Routes registra las rutas del handler en el router Chi proporcionado.
func (h *CategoryHandler) Routes(r chi.Router) {
	r.Post("/", h.CreateCategory)
	r.Get("/", h.ListCategories)
	r.Get("/{id}", h.GetCategory)
	r.Put("/{id}", h.UpdateCategory)
	r.Delete("/{id}", h.DeleteCategory)
}

// CreateCategory godoc
//
//	@Summary		Crear categoría
//	@Description	Crea una nueva categoría temática. Requiere autenticación JWT.
//	@Tags			categorías
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		service.CreateCategoryInput	true	"Datos de la categoría"
//	@Success		201		{object}	models.Category
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/categories [post]
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var input service.CreateCategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido: "+err.Error())
		return
	}
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "el campo name es requerido")
		return
	}
	cat, err := h.svc.CreateCategory(r.Context(), input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateCategory: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, cat)
}

// ListCategories godoc
//
//	@Summary		Listar categorías
//	@Description	Devuelve la lista de todas las categorías. Requiere autenticación JWT.
//	@Tags			categorías
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		models.Category
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/categories [get]
func (h *CategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	pg, err := h.svc.ListCategories(r.Context(), repository.PageParams{Page: 1, PageSize: 200})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListCategories: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, pg.Items)
}

// GetCategory godoc
//
//	@Summary		Obtener categoría por ID
//	@Description	Devuelve una categoría por su UUID. Requiere autenticación JWT.
//	@Tags			categorías
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"UUID de la categoría"	format(uuid)
//	@Success		200	{object}	models.Category
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/categories/{id} [get]
func (h *CategoryHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UUID inválido")
		return
	}
	cat, err := h.svc.GetCategory(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "GetCategory: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, cat)
}

// UpdateCategory godoc
//
//	@Summary		Actualizar categoría
//	@Description	Actualiza el nombre de una categoría existente. Requiere autenticación JWT.
//	@Tags			categorías
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"UUID de la categoría"	format(uuid)
//	@Param			body	body		service.UpdateCategoryInput	true	"Campos a actualizar"
//	@Success		200		{object}	models.Category
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UUID inválido")
		return
	}
	var input service.UpdateCategoryInput
	if err = json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido: "+err.Error())
		return
	}
	cat, err := h.svc.UpdateCategory(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "UpdateCategory: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, cat)
}

// DeleteCategory godoc
//
//	@Summary		Eliminar categoría
//	@Description	Elimina una categoría por su UUID. Requiere autenticación JWT.
//	@Tags			categorías
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"UUID de la categoría"	format(uuid)
//	@Success		204	"Sin contenido"
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UUID inválido")
		return
	}
	if err = h.svc.DeleteCategory(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "DeleteCategory: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
