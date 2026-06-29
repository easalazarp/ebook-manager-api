package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// AuthAPIHandler expone endpoints REST para autenticación (login / register).
// Devuelve JSON en todas las respuestas; no usa cookies ni redireccionamientos.
type AuthAPIHandler struct {
	supabaseURL string
	anonKey     string
}

// NewAuthAPIHandler crea un nuevo AuthAPIHandler.
func NewAuthAPIHandler(supabaseURL, anonKey string) *AuthAPIHandler {
	return &AuthAPIHandler{supabaseURL: supabaseURL, anonKey: anonKey}
}

// ─── Request / Response DTOs ────────────────────────────────────────────────

// LoginRequest es el body esperado por el endpoint de login.
type LoginRequest struct {
	// Correo electrónico del usuario.
	// example: user@example.com
	Email string `json:"email" example:"user@example.com"`
	// Contraseña del usuario.
	// example: password123
	Password string `json:"password" example:"password123"`
}

// RegisterRequest es el body esperado por el endpoint de registro.
type RegisterRequest struct {
	// Correo electrónico del nuevo usuario.
	// example: newuser@example.com
	Email string `json:"email" example:"newuser@example.com"`
	// Contraseña del nuevo usuario (mínimo 6 caracteres según Supabase).
	// example: password123
	Password string `json:"password" example:"password123"`
}

// AuthResponse es la respuesta devuelta tras un login exitoso.
type AuthResponse struct {
	// JWT de acceso para autenticar las siguientes peticiones.
	// Inclúyelo como header: Authorization: Bearer <token>
	AccessToken string `json:"access_token" example:"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9..."`
	// Tipo de token (siempre "bearer").
	TokenType string `json:"token_type" example:"bearer"`
	// Segundos hasta que el token expira.
	ExpiresIn int `json:"expires_in" example:"3600"`
}

// RegisterResponse es la respuesta devuelta tras un registro exitoso.
type RegisterResponse struct {
	// Mensaje informativo.
	Message string `json:"message" example:"Usuario registrado correctamente. Revisa tu email para confirmar la cuenta."`
}

// ─── Handlers ───────────────────────────────────────────────────────────────

// Login godoc
//
//	@Summary		Iniciar sesión
//	@Description	Autentica al usuario con email y contraseña mediante Supabase Auth.
//	@Description	Devuelve un JWT (access_token) que debe usarse en el header
//	@Description	`Authorization: Bearer <token>` para los endpoints protegidos.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginRequest	true	"Credenciales de acceso"
//	@Success		200		{object}	AuthResponse
//	@Failure		400		{object}	map[string]string	"Body inválido o campos vacíos"
//	@Failure		401		{object}	map[string]string	"Credenciales incorrectas"
//	@Failure		500		{object}	map[string]string	"Error interno"
//	@Router			/auth/login [post]
func (h *AuthAPIHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido: "+err.Error())
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email y password son requeridos")
		return
	}

	body, _ := json.Marshal(map[string]string{
		"email":    req.Email,
		"password": req.Password,
	})

	apiReq, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodPost,
		h.supabaseURL+"/auth/v1/token?grant_type=password",
		bytes.NewReader(body),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error interno")
		return
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Header.Set("apikey", h.anonKey)

	resp, err := http.DefaultClient.Do(apiReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo conectar con el servicio de autenticación")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
			Msg              string `json:"msg"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody.ErrorDescription
		if msg == "" {
			msg = errBody.Error
		}
		if msg == "" {
			msg = errBody.Msg
		}
		if msg == "" {
			msg = fmt.Sprintf("autenticación fallida (status %d)", resp.StatusCode)
		}
		writeError(w, http.StatusUnauthorized, msg)
		return
	}

	var supaResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&supaResp); err != nil || supaResp.AccessToken == "" {
		writeError(w, http.StatusInternalServerError, "respuesta inesperada del servidor de autenticación")
		return
	}

	tokenType := supaResp.TokenType
	if tokenType == "" {
		tokenType = "bearer"
	}
	expiresIn := supaResp.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 3600
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		AccessToken: supaResp.AccessToken,
		TokenType:   tokenType,
		ExpiresIn:   expiresIn,
	})
}

// Register godoc
//
//	@Summary		Registrar usuario
//	@Description	Crea una nueva cuenta de usuario en Supabase Auth.
//	@Description	Dependiendo de la configuración de Supabase, puede requerir confirmación por email.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RegisterRequest		true	"Datos del nuevo usuario"
//	@Success		201		{object}	RegisterResponse
//	@Failure		400		{object}	map[string]string	"Body inválido o campos vacíos"
//	@Failure		422		{object}	map[string]string	"Email ya registrado u otro error de validación"
//	@Failure		500		{object}	map[string]string	"Error interno"
//	@Router			/auth/register [post]
func (h *AuthAPIHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido: "+err.Error())
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email y password son requeridos")
		return
	}

	body, _ := json.Marshal(map[string]string{
		"email":    req.Email,
		"password": req.Password,
	})

	apiReq, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodPost,
		h.supabaseURL+"/auth/v1/signup",
		bytes.NewReader(body),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error interno")
		return
	}
	apiReq.Header.Set("Content-Type", "application/json")
	apiReq.Header.Set("apikey", h.anonKey)

	resp, err := http.DefaultClient.Do(apiReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no se pudo conectar con el servicio de autenticación")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errBody struct {
			Msg     string `json:"msg"`
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody.Msg
		if msg == "" {
			msg = errBody.Message
		}
		if msg == "" {
			msg = fmt.Sprintf("registro fallido (status %d)", resp.StatusCode)
		}
		writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	writeJSON(w, http.StatusCreated, RegisterResponse{
		Message: "Usuario registrado correctamente. Revisa tu email para confirmar la cuenta.",
	})
}
