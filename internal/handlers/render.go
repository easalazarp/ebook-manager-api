package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
)

// templateFuncs son las funciones de utilidad disponibles en todos los templates.
// Se definen aquí para evitar duplicación y mantener la lógica de presentación
// separada del código de negocio.
var templateFuncs = template.FuncMap{
	"contains": strings.Contains,
	"add":      func(a, b int) int { return a + b },
	"sub":      func(a, b int) int { return a - b },
	"mul":      func(a, b int) int { return a * b },
	"dict": func(pairs ...any) (map[string]any, error) {
		if len(pairs)%2 != 0 {
			return nil, fmt.Errorf("dict: número impar de argumentos")
		}
		m := make(map[string]any, len(pairs)/2)
		for i := 0; i < len(pairs); i += 2 {
			key, ok := pairs[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict: clave no es string")
			}
			m[key] = pairs[i+1]
		}
		return m, nil
	},
}

// render ejecuta el template layout.html + la vista específica `name`.
// Usa templateFuncs para exponer helpers de presentación a todos los templates.
//
// Análisis de escape: `data` se pasa como referencia hacia abajo a ExecuteTemplate.
// html/template escapa automáticamente los valores para prevenir XSS.
func render(w http.ResponseWriter, name string, data any) {
	tmpl, err := template.New("").Funcs(templateFuncs).ParseFiles(
		filepath.Join("internal", "views", "layout.html"),
		filepath.Join("internal", "views", name),
	)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err = tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
