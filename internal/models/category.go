package models

// Category representa una categoría temática para clasificar e-books.
type Category struct {
	BaseModel
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// GetSlug devuelve el slug de la categoría.
func (c *Category) GetSlug() string { return c.Slug }
