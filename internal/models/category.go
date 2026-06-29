package models

// Category representa una categoría temática para clasificar e-books.
type Category struct {
	BaseModel
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// NewCategory construye una Category con UUID y timestamps generados.
func NewCategory(name, slug string) *Category {
	now := timeNow()
	return &Category{
		BaseModel: BaseModel{ID: newUUID(), CreatedAt: now, UpdatedAt: now},
		Name:      name,
		Slug:      slug,
	}
}

// GetSlug devuelve el slug de la categoría.
func (c *Category) GetSlug() string { return c.Slug }
