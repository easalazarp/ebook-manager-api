package config

import (
	"fmt"
	"os"
)

// Config holds all environment-based configuration for the server.
type Config struct {
	Port                   string
	DatabaseURL            string
	SupabaseJWTSecret      string
	SupabaseURL            string
	SupabaseAnonKey        string
	SupabaseServiceRoleKey string // Necesario para operaciones de Storage con permisos de admin
	SupabaseBucket         string // Nombre del bucket de Storage donde se almacenan los e-books
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	required := map[string]*string{
		"DATABASE_URL":              new(string),
		"SUPABASE_JWT_SECRET":       new(string),
		"SUPABASE_URL":              new(string),
		"SUPABASE_ANON_KEY":         new(string),
		"SUPABASE_SERVICE_ROLE_KEY": new(string),
	}
	for k, ptr := range required {
		v := os.Getenv(k)
		if v == "" {
			return nil, fmt.Errorf("missing required environment variable: %s", k)
		}
		*ptr = v
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	bucket := os.Getenv("SUPABASE_BUCKET")
	if bucket == "" {
		bucket = "ebooks"
	}

	return &Config{
		Port:                   port,
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		SupabaseJWTSecret:      os.Getenv("SUPABASE_JWT_SECRET"),
		SupabaseURL:            os.Getenv("SUPABASE_URL"),
		SupabaseAnonKey:        os.Getenv("SUPABASE_ANON_KEY"),
		SupabaseServiceRoleKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		SupabaseBucket:         bucket,
	}, nil
}
