package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type contextKey string

const userIDKey contextKey = "userID"

// NewJWKSCache carga el conjunto de claves públicas desde el endpoint JWKS de Supabase.
// Debe llamarse una vez al arrancar el servidor.
func NewJWKSCache(supabaseURL string) (jwk.Set, error) {
	url := supabaseURL + "/auth/v1/.well-known/jwks.json"
	set, err := jwk.Fetch(context.Background(), url)
	if err != nil {
		return nil, fmt.Errorf("middleware: fetch JWKS from %s: %w", url, err)
	}
	return set, nil
}

func parseToken(tokenStr string, keySet jwk.Set) (jwt.Token, error) {
	return jwt.Parse([]byte(tokenStr),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
	)
}

func uidFromToken(token jwt.Token) uuid.UUID {
	uid, _ := uuid.Parse(token.Subject())
	return uid
}

// RequireAuth valida el JWT de la cookie "token". Redirige a /login si inválido.
func RequireAuth(keySet jwk.Set) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			token, err := parseToken(cookie.Value, keySet)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, uidFromToken(token))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extrae el userID inyectado por RequireAuth.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	uid, ok := ctx.Value(userIDKey).(uuid.UUID)
	return uid, ok && uid != uuid.Nil
}

// RequireAuthAPI valida el JWT para rutas de la API REST.
// Acepta token en cookie "token" o header "Authorization: Bearer <token>".
// Responde 401 JSON en lugar de redirigir.
func RequireAuthAPI(keySet jwk.Set) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""
			if cookie, err := r.Cookie("token"); err == nil {
				tokenStr = cookie.Value
			} else if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
				tokenStr = h[7:]
			}

			unauth := func(msg string) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
			}

			if tokenStr == "" {
				unauth("autenticación requerida")
				return
			}

			token, err := parseToken(tokenStr, keySet)
			if err != nil {
				unauth("token inválido o expirado")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, uidFromToken(token))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
