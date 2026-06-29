package repository

// PageParams contiene los parámetros de paginación para cualquier consulta de lista.
type PageParams struct {
	Page     int // 1-based; 1 = primera página
	PageSize int // registros por página (10, 30 o 50)
}

// Clamp normaliza los valores de paginación para evitar valores inválidos.
// - Page mínimo: 1
// - PageSize válidos: 10, 30, 50 (default: 10 si el valor no es uno de esos)
func (p PageParams) Clamp() PageParams {
	if p.Page < 1 {
		p.Page = 1
	}
	switch p.PageSize {
	case 10, 30, 50:
		// válido
	default:
		p.PageSize = 10
	}
	return p
}

// Offset calcula el OFFSET SQL a partir de Page y PageSize.
func (p PageParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Page es un resultado paginado genérico.
type Page[T any] struct {
	Items       []T
	Total       int // total de registros sin paginación
	PageSize    int
	CurrentPage int
	TotalPages  int
}

// NewPage construye un Page calculando TotalPages automáticamente.
func NewPage[T any](items []T, total, page, pageSize int) Page[T] {
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	return Page[T]{
		Items:       items,
		Total:       total,
		PageSize:    pageSize,
		CurrentPage: page,
		TotalPages:  totalPages,
	}
}

// HasPrev informa si hay una página anterior.
func (p Page[T]) HasPrev() bool { return p.CurrentPage > 1 }

// HasNext informa si hay una página siguiente.
func (p Page[T]) HasNext() bool { return p.CurrentPage < p.TotalPages }

// IsFirst informa si es la primera página.
func (p Page[T]) IsFirst() bool { return p.CurrentPage == 1 }

// IsLast informa si es la última página.
func (p Page[T]) IsLast() bool { return p.CurrentPage >= p.TotalPages }

// FirstPage retorna siempre 1.
func (p Page[T]) FirstPage() int { return 1 }

// LastPage retorna el número de la última página.
func (p Page[T]) LastPage() int { return p.TotalPages }

// PrevPage retorna el número de la página anterior (mínimo 1).
func (p Page[T]) PrevPage() int {
	if p.CurrentPage > 1 {
		return p.CurrentPage - 1
	}
	return 1
}

// NextPage retorna el número de la página siguiente.
func (p Page[T]) NextPage() int {
	if p.CurrentPage < p.TotalPages {
		return p.CurrentPage + 1
	}
	return p.TotalPages
}

// Pages retorna una lista de números de página para renderizar el paginador.
// Muestra máximo 7 páginas: siempre la primera, la última y hasta 5 alrededor de la actual.
func (p Page[T]) Pages() []int {
	if p.TotalPages <= 7 {
		pages := make([]int, p.TotalPages)
		for i := range pages {
			pages[i] = i + 1
		}
		return pages
	}
	// ventana de 5 alrededor de la página actual
	start := p.CurrentPage - 2
	end := p.CurrentPage + 2
	if start < 1 {
		start = 1
		end = 5
	}
	if end > p.TotalPages {
		end = p.TotalPages
		start = end - 4
	}
	pages := []int{1}
	if start > 2 {
		pages = append(pages, -1) // separador "..."
	}
	for i := start; i <= end; i++ {
		if i != 1 && i != p.TotalPages {
			pages = append(pages, i)
		}
	}
	if end < p.TotalPages-1 {
		pages = append(pages, -1) // separador "..."
	}
	pages = append(pages, p.TotalPages)
	return pages
}
