package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// AuthHandler maneja las vistas y acciones de autenticación via Supabase Auth.
type AuthHandler struct {
	supabaseURL string
	anonKey     string
}

func NewAuthHandler(supabaseURL, anonKey string) *AuthHandler {
	return &AuthHandler{supabaseURL: supabaseURL, anonKey: anonKey}
}

func (h *AuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	render(w, "login.html", nil)
}

func (h *AuthHandler) ShowRegister(w http.ResponseWriter, r *http.Request) {
	render(w, "register.html", nil)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost,
		h.supabaseURL+"/auth/v1/token?grant_type=password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", h.anonKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		render(w, "login.html", map[string]string{"Error": "Credenciales inválidas"})
		return
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil || result.AccessToken == "" {
		render(w, "login.html", map[string]string{"Error": "Credenciales inválidas"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    result.AccessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost,
		h.supabaseURL+"/auth/v1/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", h.anonKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		render(w, "register.html", map[string]string{"Error": "No se pudo registrar"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errBody struct {
			Msg string `json:"msg"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody.Msg
		if msg == "" {
			msg = fmt.Sprintf("No se pudo registrar (status %d)", resp.StatusCode)
		}
		render(w, "register.html", map[string]string{"Error": msg})
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}
