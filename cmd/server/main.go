// Package main es el punto de entrada del servidor.
//
//	@title			E-Book Management API
//	@version		1.0
//	@description	API REST para gestión de e-books, autores y categorías.
//	@description	Usa POST /api/v1/auth/login para obtener un token JWT y luego
//	@description	inclúyelo como `Authorization: Bearer <token>` en las peticiones protegidas.
//
//	@contact.name	Soporte
//	@contact.email	soporte@example.com
//
//	@license.name	MIT
//
//	@host		localhost:8080
//	@BasePath	/api/v1
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT obtenido de POST /api/v1/auth/login. Formato: `Bearer <token>`
package main

import (
	"context"
	"fmt"
	"net/http"

	// pprof: registra automáticamente sus handlers al importarse con el efecto lateral (_).
	// Los endpoints /debug/pprof/* solo se montan en el servidor de diagnóstico interno,
	// NUNCA en el servidor público, para evitar exponer información sensible de memoria/CPU.
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/yourorg/ebook-management-backend/docs" // generado por swag
	"github.com/yourorg/ebook-management-backend/internal/config"
	"github.com/yourorg/ebook-management-backend/internal/handlers"
	appmiddleware "github.com/yourorg/ebook-management-backend/internal/middleware"
	"github.com/yourorg/ebook-management-backend/internal/repository"
	"github.com/yourorg/ebook-management-backend/internal/service"
	"github.com/yourorg/ebook-management-backend/internal/storage"
)

func main() {
	// 1. Config — carga todas las variables de entorno requeridas
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	// 2. Cargar claves públicas JWKS desde Supabase (soporta ES256)
	// Se obtienen en startup; en producción deberías refrescarlas periódicamente.
	keySet, err := appmiddleware.NewJWKSCache(cfg.SupabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "jwks error: %v\n", err)
		os.Exit(1)
	}

	// 3. DB pool — pgxpool gestiona un pool de conexiones automáticamente.
	// QueryExecModeSimpleProtocol asegura compatibilidad con PgBouncer (Supabase usa PgBouncer
	// en el puerto 6543). El pool reutiliza conexiones, reduciendo latencia y overhead de TLS.
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db config error: %v\n", err)
		os.Exit(1)
	}
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// MaxConns y MinConns del pool — valores conservadores para Supabase Free Tier.
	// Ajustar según el plan de BD y la carga esperada.
	poolCfg.MaxConns = 10
	poolCfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db connect error: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err = pool.Ping(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "db ping error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Base de datos conectada")

	// 4. Storage client — cliente para Supabase Storage (subida de archivos, signed URLs).
	// Usa service_role key para operaciones administrativas desde el servidor.
	storageClient := storage.NewClient(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey, cfg.SupabaseBucket)
	fmt.Printf("✓ Storage configurado (bucket: %s)\n", cfg.SupabaseBucket)

	// 5. Repositories — implementaciones concretas con pgxpool.
	// Se inyectan como interfaces en los servicios para facilitar testing con mocks.
	ebookRepo := repository.NewPostgresEBookRepository(pool)
	authorRepo := repository.NewPostgresAuthorRepository(pool)
	logRepo := repository.NewPostgresLogRepository(pool)
	categoryRepo := repository.NewPostgresCategoryRepository(pool)

	// 6. Services — lógica de negocio, composición de dependencias.
	logService := service.NewLogServiceImpl(logRepo)
	ebookService := service.NewEBookServiceImpl(ebookRepo, logService)
	authorService := service.NewAuthorServiceImpl(authorRepo)
	categoryService := service.NewCategoryServiceImpl(categoryRepo)

	// 7. Handlers — capa HTTP, sin lógica de negocio.
	ebookHandler := handlers.NewEBookHandler(ebookService)
	authorHandler := handlers.NewAuthorHandler(authorService)
	authHandler := handlers.NewAuthHandler(cfg.SupabaseURL, cfg.SupabaseAnonKey)
	authAPIHandler := handlers.NewAuthAPIHandler(cfg.SupabaseURL, cfg.SupabaseAnonKey)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	uploadHandler := handlers.NewUploadHandler(ebookService, storageClient)
	webHandler := handlers.NewWebHandler(ebookService, logService, authorService, storageClient)
	adminHandler := handlers.NewAdminHandler(ebookService, authorService, categoryService, logService, storageClient)

	// 8. Router principal (servidor público)
	r := chi.NewRouter()
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(chimiddleware.Recoverer)
	// Logger de requests — útil en desarrollo para ver el flujo de peticiones
	r.Use(chimiddleware.Logger)

	// Archivos estáticos (CSS, imágenes locales, etc.)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Health check — para balanceadores de carga y monitoreo externo
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// --- Rutas MVC públicas ---
	r.Get("/", webHandler.ListCatalog)
	r.Get("/login", authHandler.ShowLogin)
	r.Post("/login", authHandler.Login)
	r.Get("/register", authHandler.ShowRegister)
	r.Post("/register", authHandler.Register)
	r.Post("/logout", authHandler.Logout)
	r.Get("/ebooks/{id}", webHandler.ShowEBook)
	r.Get("/authors", webHandler.ListAuthorsWeb)

	// --- Rutas MVC protegidas (usuario autenticado) ---
	// Descarga protegida: solo usuarios con sesión activa pueden bajar los archivos.
	r.With(appmiddleware.RequireAuth(keySet)).
		Get("/ebooks/{id}/download", webHandler.DownloadEBook)

	// --- Panel de administración (protegido por RequireAuth) ---
	// Todas las rutas bajo /admin requieren JWT válido en cookie.
	// RequireAuth redirige a /login si el token no es válido o está ausente.
	r.Route("/admin", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(keySet))
		adminHandler.Routes(r)
	})

	// --- REST API pública: auth ---
	r.Post("/api/v1/auth/login", authAPIHandler.Login)
	r.Post("/api/v1/auth/register", authAPIHandler.Register)

	// --- Swagger UI ---
	// Accesible en http://localhost:8080/swagger/index.html
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// --- REST API (protegida, acepta cookie o Bearer token) ---
	r.Route("/api/v1/ebooks", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuthAPI(keySet))
		ebookHandler.Routes(r)
		// Endpoints de subida de archivos (mismo grupo /api/v1/ebooks)
		r.Post("/{id}/cover", uploadHandler.UploadCover)
		r.Post("/{id}/file", uploadHandler.UploadFile)
	})
	r.Route("/api/v1/authors", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuthAPI(keySet))
		authorHandler.Routes(r)
	})
	r.Route("/api/v1/categories", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuthAPI(keySet))
		categoryHandler.Routes(r)
	})
	r.With(appmiddleware.RequireAuthAPI(keySet)).
		Get("/api/v1/ebooks/{id}/author", authorHandler.GetAuthorByBookID)

	// 9. Servidor de diagnóstico interno con pprof
	//
	// IMPORTANTE DE SEGURIDAD: pprof se monta en un servidor SEPARADO en el puerto 6060.
	// Nunca debe exponerse en el servidor público (puerto 8080) porque:
	//  - /debug/pprof/heap   → revela estructura de memoria interna
	//  - /debug/pprof/goroutine → revela goroutines activas (concurrencia)
	//  - /debug/pprof/profile → genera CPU profiles (afecta performance)
	//
	// En producción, este servidor solo debe ser accesible desde localhost o VPN interna.
	// Para generar un Flame Graph de CPU: go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
	// Para analizar fugas de memoria en el heap: go tool pprof http://localhost:6060/debug/pprof/heap
	pprofAddr := ":6060"
	pprofSrv := &http.Server{
		Addr:         pprofAddr,
		Handler:      http.DefaultServeMux, // pprof registra en http.DefaultServeMux al importarse
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // Los perfiles de CPU pueden tardar hasta 30s+
	}
	go func() {
		fmt.Printf("✓ Servidor pprof en %s (solo acceso local)\n", pprofAddr)
		if err := pprofSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "pprof server error: %v\n", err)
		}
	}()

	// 10. HTTP server principal con timeouts correctamente configurados.
	// WriteTimeout > ReadTimeout para dar tiempo a las subidas multipart.
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 75 * time.Second, // >= maxUploadSize / ancho de banda mínimo esperado
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("✓ Servidor principal en :%s\n", cfg.Port)

	// 11. Graceful shutdown — espera hasta 10s para que las requests en vuelo terminen.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-quit
	fmt.Println("\nApagando servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
	}

	// También apagar el servidor de diagnóstico
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = pprofSrv.Shutdown(shutCtx)

	fmt.Println("Servidor detenido correctamente.")
}
