package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yourorg/ebook-management-backend/internal/middleware"
	"github.com/yourorg/ebook-management-backend/internal/models"
	"github.com/yourorg/ebook-management-backend/internal/repository"
	"github.com/yourorg/ebook-management-backend/internal/service"
	"github.com/yourorg/ebook-management-backend/internal/storage"
)

const maxUploadSize = 50 << 20 // 50 MB
const maxCoverSize = 5 << 20   // 5 MB

var mimeByFormat = map[string]string{
	models.FormatPDF:  "application/pdf",
	models.FormatEPUB: "application/epub+zip",
	models.FormatMOBI: "application/x-mobipocket-ebook",
}

var coverMIME = map[string]string{
	".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".png": "image/png", ".webp": "image/webp",
}

// AdminHandler gestiona el panel de administración protegido.
type AdminHandler struct {
	bookSvc       service.IBookService
	authorSvc     service.IAuthorService
	categorySvc   service.ICategoryService
	logSvc        service.ILogService
	storageClient *storage.Client
}

func NewAdminHandler(
	bookSvc service.IBookService,
	authorSvc service.IAuthorService,
	categorySvc service.ICategoryService,
	logSvc service.ILogService,
	sc *storage.Client,
) *AdminHandler {
	return &AdminHandler{
		bookSvc: bookSvc, authorSvc: authorSvc,
		categorySvc: categorySvc, logSvc: logSvc, storageClient: sc,
	}
}

func (h *AdminHandler) Routes(r chi.Router) {
	r.Get("/", h.Dashboard)

	r.Get("/ebooks/new", h.ShowCreateEBook)
	r.Post("/ebooks", h.CreateEBook)
	r.Get("/ebooks/{id}/edit", h.ShowEditEBook)
	r.Post("/ebooks/{id}", h.UpdateEBook)
	r.Post("/ebooks/{id}/delete", h.DeleteEBook)

	r.Get("/authors/new", h.ShowCreateAuthor)
	r.Post("/authors", h.CreateAuthor)
	r.Get("/authors/{id}/edit", h.ShowEditAuthor)
	r.Post("/authors/{id}", h.UpdateAuthor)
	r.Post("/authors/{id}/delete", h.DeleteAuthor)

	r.Get("/categories/new", h.ShowCreateCategory)
	r.Post("/categories", h.CreateCategory)
	r.Get("/categories/{id}/edit", h.ShowEditCategory)
	r.Post("/categories/{id}", h.UpdateCategory)
	r.Post("/categories/{id}/delete", h.DeleteCategory)

	r.Get("/logs", h.ShowLogs)
}

// ── helpers de paginación ────────────────────────────────────────────────

func pageParams(r *http.Request) repository.PageParams {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	return repository.PageParams{Page: page, PageSize: size}
}

// ── Dashboard ────────────────────────────────────────────────────────────

type dashboardData struct {
	EBooks     repository.Page[*models.EBook]
	Authors    repository.Page[*models.Author]
	Categories repository.Page[*models.Category]
	UserID     bool
	Flash      string
	// tab activo para que el redirect mantenga el tab correcto
	ActiveTab string
}

func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	p := pageParams(r)
	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = "ebooks"
	}

	ebooks, err := h.bookSvc.ListEBooks(r.Context(), p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AdminHandler.Dashboard: ListEBooks: %v\n", err)
		http.Error(w, "Error al cargar el panel.", http.StatusInternalServerError)
		return
	}
	if h.storageClient != nil {
		for _, eb := range ebooks.Items {
			if eb.CoverURL != "" {
				eb.CoverURL = h.storageClient.PublicURL(eb.CoverURL)
			}
		}
	}

	authors, err := h.authorSvc.ListAuthors(r.Context(), p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AdminHandler.Dashboard: ListAuthors: %v\n", err)
		http.Error(w, "Error al cargar los autores.", http.StatusInternalServerError)
		return
	}

	categories, err := h.categorySvc.ListCategories(r.Context(), p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AdminHandler.Dashboard: ListCategories: %v\n", err)
		http.Error(w, "Error al cargar las categorías.", http.StatusInternalServerError)
		return
	}

	render(w, "admin_dashboard.html", dashboardData{
		EBooks: ebooks, Authors: authors, Categories: categories,
		UserID: isAuthenticated(r), Flash: r.URL.Query().Get("flash"),
		ActiveTab: tab,
	})
}

// ── E-Books: crear ───────────────────────────────────────────────────────

type createEBookData struct {
	Authors []repository.Page[*models.Author]
	All     []*models.Author
	UserID  bool
	Error   string
}

func (h *AdminHandler) ShowCreateEBook(w http.ResponseWriter, r *http.Request) {
	authors := h.allAuthors(r)
	render(w, "admin_create_ebook.html", struct {
		Authors []*models.Author
		UserID  bool
		Error   string
	}{authors, isAuthenticated(r), ""})
}

func (h *AdminHandler) CreateEBook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+maxCoverSize+4096)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		h.renderCreateEBookError(w, r, "El formulario supera el tamaño máximo permitido.")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	authorIDStr := strings.TrimSpace(r.FormValue("author_id"))
	format := strings.TrimSpace(r.FormValue("format"))

	if title == "" || authorIDStr == "" || format == "" {
		h.renderCreateEBookError(w, r, "Título, autor y formato son obligatorios.")
		return
	}
	authorID, err := uuid.Parse(authorIDStr)
	if err != nil {
		h.renderCreateEBookError(w, r, "El autor seleccionado no es válido.")
		return
	}
	if err = models.ValidateFormat(format); err != nil {
		h.renderCreateEBookError(w, r, "Formato no soportado. Usa: epub, pdf o mobi.")
		return
	}

	// Portada (opcional)
	var coverPath string
	coverFile, coverHeader, coverErr := r.FormFile("cover")
	if coverErr == nil {
		defer coverFile.Close()
		if coverHeader.Size > maxCoverSize {
			h.renderCreateEBookError(w, r, "La portada no puede superar los 5 MB.")
			return
		}
		ext := strings.ToLower(filepath.Ext(coverHeader.Filename))
		ct, ok := coverMIME[ext]
		if !ok {
			h.renderCreateEBookError(w, r, "Formato de portada no soportado. Usa JPG, PNG o WebP.")
			return
		}
		coverPath, err = h.storageClient.Upload(r.Context(),
			fmt.Sprintf("covers/%s%s", uuid.New(), ext), ct, coverFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "AdminHandler.CreateEBook: cover upload: %v\n", err)
			h.renderCreateEBookError(w, r, "No se pudo subir la portada.")
			return
		}
	}

	// Archivo del libro
	bookFile, bookHeader, err := r.FormFile("file")
	if err != nil {
		if coverPath != "" {
			_ = h.storageClient.Delete(r.Context(), coverPath)
		}
		h.renderCreateEBookError(w, r, "Debes seleccionar un archivo para subir.")
		return
	}
	defer bookFile.Close()

	ct := mimeByFormat[format]
	if ct == "" {
		ct = bookHeader.Header.Get("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}
	}
	filePath := fmt.Sprintf("%s/%s.%s", format, uuid.New(), format)
	fileURL, err := h.storageClient.Upload(r.Context(), filePath, ct, bookFile)
	if err != nil {
		if coverPath != "" {
			_ = h.storageClient.Delete(r.Context(), coverPath)
		}
		fmt.Fprintf(os.Stderr, "AdminHandler.CreateEBook: book upload: %v\n", err)
		h.renderCreateEBookError(w, r, "No se pudo subir el archivo.")
		return
	}

	created, err := h.bookSvc.CreateEBook(r.Context(), service.CreateEBookInput{
		AuthorID: authorID, Title: title, Description: description,
		CoverURL: coverPath,
		Files:    []service.FileInput{{Format: format, FileURL: fileURL}},
	})
	if err != nil {
		_ = h.storageClient.Delete(r.Context(), filePath)
		if coverPath != "" {
			_ = h.storageClient.Delete(r.Context(), coverPath)
		}
		fmt.Fprintf(os.Stderr, "AdminHandler.CreateEBook: service: %v\n", err)
		h.renderCreateEBookError(w, r, "No se pudo guardar el e-book.")
		return
	}

	if uid, ok := middleware.UserIDFromContext(r.Context()); ok {
		_ = h.logSvc.RecordActivity(r.Context(), fmt.Sprintf("admin_ebook_created:%s", created.ID), uid)
	}
	http.Redirect(w, r, "/admin?flash=E-book+creado+exitosamente&tab=ebooks", http.StatusSeeOther)
}

// ── E-Books: editar ──────────────────────────────────────────────────────

type editEBookData struct {
	EBook   *models.EBook
	Authors []*models.Author
	UserID  bool
	Error   string
}

func (h *AdminHandler) ShowEditEBook(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ebook, err := h.bookSvc.GetEBook(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if ebook.CoverURL != "" && h.storageClient != nil {
		ebook.CoverURL = h.storageClient.PublicURL(ebook.CoverURL)
	}
	render(w, "admin_edit_ebook.html", editEBookData{
		EBook: ebook, Authors: h.allAuthors(r), UserID: isAuthenticated(r),
	})
}

func (h *AdminHandler) UpdateEBook(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err = r.ParseMultipartForm(maxCoverSize + 4096); err != nil {
		_ = r.ParseForm()
	}

	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	authorIDStr := strings.TrimSpace(r.FormValue("author_id"))

	if title == "" {
		h.renderEditEBookError(w, r, id, "El título es obligatorio.")
		return
	}

	input := service.UpdateEBookInput{Title: &title, Description: &description}
	if authorIDStr != "" {
		if aid, err2 := uuid.Parse(authorIDStr); err2 == nil {
			input.AuthorID = &aid
		}
	}

	// Nueva portada (opcional)
	coverFile, coverHeader, coverErr := r.FormFile("cover")
	if coverErr == nil {
		defer coverFile.Close()
		ext := strings.ToLower(filepath.Ext(coverHeader.Filename))
		ct, ok := coverMIME[ext]
		if ok && coverHeader.Size <= maxCoverSize {
			coverPath, uploadErr := h.storageClient.Upload(r.Context(),
				fmt.Sprintf("covers/%s%s", uuid.New(), ext), ct, coverFile)
			if uploadErr == nil {
				input.CoverURL = &coverPath
			} else {
				fmt.Fprintf(os.Stderr, "AdminHandler.UpdateEBook: cover upload: %v\n", uploadErr)
			}
		}
	}

	_, err = h.bookSvc.UpdateEBook(r.Context(), id, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AdminHandler.UpdateEBook: %v\n", err)
		h.renderEditEBookError(w, r, id, "No se pudo actualizar el e-book.")
		return
	}

	if uid, ok := middleware.UserIDFromContext(r.Context()); ok {
		_ = h.logSvc.RecordActivity(r.Context(), fmt.Sprintf("admin_ebook_updated:%s", id), uid)
	}
	http.Redirect(w, r, "/admin?flash=E-book+actualizado&tab=ebooks", http.StatusSeeOther)
}

func (h *AdminHandler) DeleteEBook(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido.", http.StatusBadRequest)
		return
	}
	if uid, ok := middleware.UserIDFromContext(r.Context()); ok {
		_ = h.logSvc.RecordActivity(r.Context(), fmt.Sprintf("admin_ebook_deleted:%s", id), uid)
	}
	if err = h.bookSvc.DeleteEBook(r.Context(), id); err != nil {
		if isNotFound(err) {
			http.Redirect(w, r, "/admin?flash=E-book+no+encontrado&tab=ebooks", http.StatusSeeOther)
			return
		}
		fmt.Fprintf(os.Stderr, "AdminHandler.DeleteEBook: %v\n", err)
		http.Redirect(w, r, "/admin?flash=Error+al+eliminar+el+e-book&tab=ebooks", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin?flash=E-book+eliminado&tab=ebooks", http.StatusSeeOther)
}

// ── Autores: crear ───────────────────────────────────────────────────────

type authorFormData struct {
	Author *models.Author // nil en creación
	UserID bool
	Error  string
}

func (h *AdminHandler) ShowCreateAuthor(w http.ResponseWriter, r *http.Request) {
	render(w, "admin_create_author.html", authorFormData{UserID: isAuthenticated(r)})
}

func (h *AdminHandler) CreateAuthor(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render(w, "admin_create_author.html", authorFormData{UserID: isAuthenticated(r), Error: "Error al procesar el formulario."})
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	bio := strings.TrimSpace(r.FormValue("bio"))
	if name == "" {
		render(w, "admin_create_author.html", authorFormData{UserID: isAuthenticated(r), Error: "El nombre es obligatorio."})
		return
	}
	created, err := h.authorSvc.CreateAuthor(r.Context(), service.CreateAuthorInput{Name: name, Bio: bio})
	if err != nil {
		fmt.Fprintf(os.Stderr, "AdminHandler.CreateAuthor: %v\n", err)
		render(w, "admin_create_author.html", authorFormData{UserID: isAuthenticated(r), Error: "No se pudo guardar el autor."})
		return
	}
	if uid, ok := middleware.UserIDFromContext(r.Context()); ok {
		_ = h.logSvc.RecordActivity(r.Context(), fmt.Sprintf("admin_author_created:%s", created.ID), uid)
	}
	http.Redirect(w, r, "/admin?flash=Autor+creado+exitosamente&tab=authors", http.StatusSeeOther)
}

// ── Autores: editar ──────────────────────────────────────────────────────

func (h *AdminHandler) ShowEditAuthor(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	author, err := h.authorSvc.GetAuthor(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	render(w, "admin_edit_author.html", authorFormData{Author: author, UserID: isAuthenticated(r)})
}

func (h *AdminHandler) UpdateAuthor(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err = r.ParseForm(); err != nil {
		http.Error(w, "Error al procesar el formulario.", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	bio := strings.TrimSpace(r.FormValue("bio"))
	if name == "" {
		author, _ := h.authorSvc.GetAuthor(r.Context(), id)
		render(w, "admin_edit_author.html", authorFormData{Author: author, UserID: isAuthenticated(r), Error: "El nombre es obligatorio."})
		return
	}
	_, err = h.authorSvc.UpdateAuthor(r.Context(), id, service.UpdateAuthorInput{Name: &name, Bio: &bio})
	if err != nil {
		fmt.Fprintf(os.Stderr, "AdminHandler.UpdateAuthor: %v\n", err)
		author, _ := h.authorSvc.GetAuthor(r.Context(), id)
		render(w, "admin_edit_author.html", authorFormData{Author: author, UserID: isAuthenticated(r), Error: "No se pudo actualizar el autor."})
		return
	}
	if uid, ok := middleware.UserIDFromContext(r.Context()); ok {
		_ = h.logSvc.RecordActivity(r.Context(), fmt.Sprintf("admin_author_updated:%s", id), uid)
	}
	http.Redirect(w, r, "/admin?flash=Autor+actualizado&tab=authors", http.StatusSeeOther)
}

func (h *AdminHandler) DeleteAuthor(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido.", http.StatusBadRequest)
		return
	}
	if uid, ok := middleware.UserIDFromContext(r.Context()); ok {
		_ = h.logSvc.RecordActivity(r.Context(), fmt.Sprintf("admin_author_deleted:%s", id), uid)
	}
	if err = h.authorSvc.DeleteAuthor(r.Context(), id); err != nil {
		if isNotFound(err) {
			http.Redirect(w, r, "/admin?flash=Autor+no+encontrado&tab=authors", http.StatusSeeOther)
			return
		}
		fmt.Fprintf(os.Stderr, "AdminHandler.DeleteAuthor: %v\n", err)
		http.Redirect(w, r, "/admin?flash=Error+al+eliminar+autor&tab=authors", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin?flash=Autor+eliminado&tab=authors", http.StatusSeeOther)
}

// ── Categorías: crear ────────────────────────────────────────────────────

type categoryFormData struct {
	Category *models.Category // nil en creación
	UserID   bool
	Error    string
}

func (h *AdminHandler) ShowCreateCategory(w http.ResponseWriter, r *http.Request) {
	render(w, "admin_create_category.html", categoryFormData{UserID: isAuthenticated(r)})
}

func (h *AdminHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render(w, "admin_create_category.html", categoryFormData{UserID: isAuthenticated(r), Error: "Error al procesar el formulario."})
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		render(w, "admin_create_category.html", categoryFormData{UserID: isAuthenticated(r), Error: "El nombre es obligatorio."})
		return
	}
	created, err := h.categorySvc.CreateCategory(r.Context(), service.CreateCategoryInput{Name: name})
	if err != nil {
		var valErr *service.ErrValidation
		if isErrValidation(err, &valErr) {
			render(w, "admin_create_category.html", categoryFormData{UserID: isAuthenticated(r), Error: valErr.Msg})
			return
		}
		fmt.Fprintf(os.Stderr, "AdminHandler.CreateCategory: %v\n", err)
		render(w, "admin_create_category.html", categoryFormData{UserID: isAuthenticated(r), Error: "No se pudo guardar la categoría."})
		return
	}
	if uid, ok := middleware.UserIDFromContext(r.Context()); ok {
		_ = h.logSvc.RecordActivity(r.Context(), fmt.Sprintf("admin_category_created:%s", created.ID), uid)
	}
	http.Redirect(w, r, "/admin?flash=Categoría+creada&tab=categories", http.StatusSeeOther)
}

// ── Categorías: editar ───────────────────────────────────────────────────

func (h *AdminHandler) ShowEditCategory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cat, err := h.categorySvc.GetCategory(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	render(w, "admin_edit_category.html", categoryFormData{Category: cat, UserID: isAuthenticated(r)})
}

func (h *AdminHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err = r.ParseForm(); err != nil {
		http.Error(w, "Error al procesar el formulario.", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		cat, _ := h.categorySvc.GetCategory(r.Context(), id)
		render(w, "admin_edit_category.html", categoryFormData{Category: cat, UserID: isAuthenticated(r), Error: "El nombre es obligatorio."})
		return
	}
	_, err = h.categorySvc.UpdateCategory(r.Context(), id, service.UpdateCategoryInput{Name: &name})
	if err != nil {
		fmt.Fprintf(os.Stderr, "AdminHandler.UpdateCategory: %v\n", err)
		cat, _ := h.categorySvc.GetCategory(r.Context(), id)
		render(w, "admin_edit_category.html", categoryFormData{Category: cat, UserID: isAuthenticated(r), Error: "No se pudo actualizar la categoría."})
		return
	}
	if uid, ok := middleware.UserIDFromContext(r.Context()); ok {
		_ = h.logSvc.RecordActivity(r.Context(), fmt.Sprintf("admin_category_updated:%s", id), uid)
	}
	http.Redirect(w, r, "/admin?flash=Categoría+actualizada&tab=categories", http.StatusSeeOther)
}

func (h *AdminHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "ID inválido.", http.StatusBadRequest)
		return
	}
	if uid, ok := middleware.UserIDFromContext(r.Context()); ok {
		_ = h.logSvc.RecordActivity(r.Context(), fmt.Sprintf("admin_category_deleted:%s", id), uid)
	}
	if err = h.categorySvc.DeleteCategory(r.Context(), id); err != nil {
		if isNotFound(err) {
			http.Redirect(w, r, "/admin?flash=Categoría+no+encontrada&tab=categories", http.StatusSeeOther)
			return
		}
		fmt.Fprintf(os.Stderr, "AdminHandler.DeleteCategory: %v\n", err)
		http.Redirect(w, r, "/admin?flash=Error+al+eliminar+categoría&tab=categories", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin?flash=Categoría+eliminada&tab=categories", http.StatusSeeOther)
}

// ── Logs ─────────────────────────────────────────────────────────────────

type logsData struct {
	Logs   []models.Log
	UserID bool
}

func (h *AdminHandler) ShowLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.logSvc.GetHistory(r.Context())
	if err != nil {
		fmt.Fprintf(os.Stderr, "AdminHandler.ShowLogs: %v\n", err)
		http.Error(w, "Error al cargar los logs.", http.StatusInternalServerError)
		return
	}
	if logs == nil {
		logs = []models.Log{}
	}
	render(w, "admin_logs.html", logsData{Logs: logs, UserID: isAuthenticated(r)})
}

// ── helpers privados ─────────────────────────────────────────────────────

// allAuthors obtiene todos los autores para poblar selectores en formularios.
func (h *AdminHandler) allAuthors(r *http.Request) []*models.Author {
	pg, err := h.authorSvc.ListAuthors(r.Context(), repository.PageParams{Page: 1, PageSize: 200})
	if err != nil {
		fmt.Fprintf(os.Stderr, "AdminHandler.allAuthors: %v\n", err)
		return []*models.Author{}
	}
	return pg.Items
}

func (h *AdminHandler) renderCreateEBookError(w http.ResponseWriter, r *http.Request, msg string) {
	render(w, "admin_create_ebook.html", struct {
		Authors []*models.Author
		UserID  bool
		Error   string
	}{h.allAuthors(r), isAuthenticated(r), msg})
}

func (h *AdminHandler) renderEditEBookError(w http.ResponseWriter, r *http.Request, id uuid.UUID, msg string) {
	ebook, _ := h.bookSvc.GetEBook(r.Context(), id)
	render(w, "admin_edit_ebook.html", editEBookData{
		EBook: ebook, Authors: h.allAuthors(r), UserID: isAuthenticated(r), Error: msg,
	})
}

func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), repository.ErrNotFound.Error())
}

func isErrValidation(err error, target **service.ErrValidation) bool {
	if v, ok := err.(*service.ErrValidation); ok {
		*target = v
		return true
	}
	return false
}
