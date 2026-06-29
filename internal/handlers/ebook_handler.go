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

// CreateEBook godoc
//
//	@Summary		Crear e-book
//	@Description	Crea un nuevo e-book. Requiere autenticación JWT.
//	@Tags			ebooks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		service.CreateEBookInput	true	"Datos del e-book"
//	@Success		201		{object}	models.EBook
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/ebooks [post]
func (h *EBookHandler) CreateEBook(w http.ResponseWriter, r *http.Request) {
	var input service.CreateEBookInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido: "+err.Error())
		return
	}

	ebook, err := h.svc.CreateEBook(r.Context(), input)
	if err != nil {
		// Distinguir errores de dominio (400) de errores internos (500)
		if models.IsErrInvalidFormat(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "CreateEBook: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, ebook)
}

// GetEBook godoc
//
//	@Summary		Obtener e-book por ID
//	@Description	Devuelve un e-book por su UUID. Requiere autenticación JWT.
//	@Tags			ebooks
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"UUID del e-book"	format(uuid)
//	@Success		200	{object}	models.EBook
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/ebooks/{id} [get]
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

// ListEBooks godoc
//
//	@Summary		Listar e-books
//	@Description	Devuelve la lista de todos los e-books. Requiere autenticación JWT.
//	@Tags			ebooks
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		models.EBook
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/ebooks [get]
func (h *EBookHandler) ListEBooks(w http.ResponseWriter, r *http.Request) {
	pg, err := h.svc.ListEBooks(r.Context(), repository.PageParams{Page: 1, PageSize: 50})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListEBooks: %v\n", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, pg.Items)
}

// UpdateEBook godoc
//
//	@Summary		Actualizar e-book
//	@Description	Actualiza los datos de un e-book existente. Requiere autenticación JWT.
//	@Tags			ebooks
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"UUID del e-book"	format(uuid)
//	@Param			body	body		service.UpdateEBookInput	true	"Campos a actualizar"
//	@Success		200		{object}	models.EBook
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/ebooks/{id} [put]
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

// DeleteEBook godoc
//
//	@Summary		Eliminar e-book
//	@Description	Elimina un e-book por su UUID. Requiere autenticación JWT.
//	@Tags			ebooks
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"UUID del e-book"	format(uuid)
//	@Success		204	"Sin contenido"
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/ebooks/{id} [delete]
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
