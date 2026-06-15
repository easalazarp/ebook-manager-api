package repository

import "errors"

// ErrNotFound se retorna cuando un recurso solicitado no existe en la BD.
var ErrNotFound = errors.New("recurso no encontrado")
