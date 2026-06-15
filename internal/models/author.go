package models

// Author representa al autor de un e-book.
type Author struct {
	BaseModel
	Name string `json:"name"`
	Bio  string `json:"bio"`
}

// GetName devuelve el nombre completo del autor.
func (a *Author) GetName() string { return a.Name }

// GetBio devuelve la biografía del autor.
func (a *Author) GetBio() string { return a.Bio }
