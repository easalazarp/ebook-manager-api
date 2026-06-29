// Package storage provee un cliente ligero para Supabase Storage.
// No usa SDK externo — interactúa directamente con la REST API de Supabase Storage
// para mantener las dependencias mínimas y el control total sobre el comportamiento.
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client es el cliente para Supabase Storage.
// Requiere la service_role key para operaciones administrativas (upload, delete, signed URLs).
// La service_role key NO debe exponerse al cliente — solo se usa en el servidor.
type Client struct {
	baseURL        string // ej: https://xxxx.supabase.co
	serviceRoleKey string // JWT de service_role
	bucket         string // nombre del bucket configurado
	httpClient     *http.Client
}

// NewClient construye un Client de Storage listo para usar.
// Se reutiliza un solo http.Client con timeout configurado para evitar goroutine leaks.
func NewClient(supabaseURL, serviceRoleKey, bucket string) *Client {
	return &Client{
		baseURL:        supabaseURL,
		serviceRoleKey: serviceRoleKey,
		bucket:         bucket,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // archivos grandes necesitan más tiempo
		},
	}
}

// Upload sube un archivo al bucket configurado.
// path es la ruta relativa dentro del bucket (ej: "pdf/uuid.pdf").
// contentType debe ser el MIME type del archivo (ej: "application/pdf").
//
// Retorna el path relativo del objeto (ej: "pdf/uuid.pdf").
// Con bucket privado, este path se usa para generar signed URLs en tiempo de acceso.
// Con bucket público, se puede construir la URL pública a partir del path.
//
// Se guarda el path (no la URL) en la BD para poder generar signed URLs flexiblemente.
func (c *Client) Upload(ctx context.Context, path, contentType string, body io.Reader) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", c.baseURL, c.bucket, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return "", fmt.Errorf("storage.Upload: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceRoleKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true") // permite reemplazar archivos existentes

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("storage.Upload: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("storage.Upload: status %d: %s", resp.StatusCode, string(b))
	}

	// Retornamos el path relativo, no una URL pública.
	// El caller decide cómo construir la URL (signed o pública) según el tipo de bucket.
	return path, nil
}

// CreateSignedURL genera una URL prefirmada temporal para descarga directa.
// expiresIn es la duración en segundos durante la cual la URL será válida.
func (c *Client) CreateSignedURL(ctx context.Context, path string, expiresIn int) (string, error) {
	apiURL := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s", c.baseURL, c.bucket, path)

	body := fmt.Sprintf(`{"expiresIn":%d}`, expiresIn)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("storage.CreateSignedURL: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("storage.CreateSignedURL: http: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("storage.CreateSignedURL: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("storage.CreateSignedURL: status %d: %s", resp.StatusCode, string(respBytes))
	}

	// Parsear con encoding/json para manejar correctamente las barras escapadas (\/).
	// La respuesta tiene la forma:
	//   {"signedURL":"/storage/v1/object/sign/<bucket>/<path>?token=..."}
	// donde las barras pueden venir escapadas como \/ en el JSON raw.
	var result struct {
		SignedURL string `json:"signedURL"`
	}
	if err = json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("storage.CreateSignedURL: parse response: %w", err)
	}
	if result.SignedURL == "" {
		return "", fmt.Errorf("storage.CreateSignedURL: signedURL vacío en respuesta: %s", string(respBytes))
	}

	// SignedURL es un path absoluto como "/storage/v1/object/sign/..."
	// Lo combinamos con el baseURL del proyecto para obtener la URL completa.
	return c.baseURL + result.SignedURL, nil
}

// Delete elimina un objeto del bucket.
// path es la ruta relativa dentro del bucket (mismo valor usado en Upload).
func (c *Client) Delete(ctx context.Context, path string) error {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", c.baseURL, c.bucket, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("storage.Delete: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceRoleKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("storage.Delete: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("storage.Delete: status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// PublicURL construye la URL pública de un objeto en el bucket.
// Solo funciona si el bucket está configurado como público en Supabase.
// Acepta tanto paths relativos ("covers/uuid.jpg") como URLs completas anteriores.
func (c *Client) PublicURL(pathOrURL string) string {
	path := c.NormalizeStoragePath(pathOrURL)
	if path == "" {
		return ""
	}
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", c.baseURL, c.bucket, path)
}

// NormalizeStoragePath extrae el path relativo dentro del bucket desde cualquier
// formato en que se haya guardado: path limpio, URL pública, URL signed, etc.
// Ver detalles en el comentario de normalizeStoragePath en web_handler.go.
func (c *Client) NormalizeStoragePath(raw string) string {
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		return raw
	}
	for _, marker := range []string{
		"/object/public/",
		"/object/sign/",
		"/object/authenticated/",
		"/object/",
	} {
		idx := strings.Index(raw, marker)
		if idx == -1 {
			continue
		}
		rest := raw[idx+len(marker):]
		if q := strings.Index(rest, "?"); q != -1 {
			rest = rest[:q]
		}
		slash := strings.Index(rest, "/")
		if slash == -1 {
			return ""
		}
		return rest[slash+1:]
	}
	return raw
}

// extractJSONString extrae el valor de una clave string de un JSON simple.
// Evita importar encoding/json para un único campo en un path caliente.
// Solo funciona para valores string simples (no anidados).
func extractJSONString(jsonStr, key string) string {
	search := `"` + key + `":"`
	idx := strings.Index(jsonStr, search)
	if idx == -1 {
		return ""
	}
	start := idx + len(search)
	end := strings.Index(jsonStr[start:], `"`)
	if end == -1 {
		return ""
	}
	return jsonStr[start : start+end]
}
