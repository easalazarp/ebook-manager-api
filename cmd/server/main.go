package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/ebook-management-backend/internal/config"
	"github.com/yourorg/ebook-management-backend/internal/handlers"
	appmiddleware "github.com/yourorg/ebook-management-backend/internal/middleware"
	"github.com/yourorg/ebook-management-backend/internal/repository"
	"github.com/yourorg/ebook-management-backend/internal/service"
)

func main() {
	// 1. Config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	// 2. Cargar claves públicas JWKS desde Supabase (soporta ES256)
	keySet, err := appmiddleware.NewJWKSCache(cfg.SupabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "jwks error: %v\n", err)
		os.Exit(1)
	}

	// 3. DB pool — deshabilita prepared statements para compatibilidad con PgBouncer (Supabase)
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db config error: %v\n", err)
		os.Exit(1)
	}
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
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

	// 4. Repositories
	ebookRepo := repository.NewPostgresEBookRepository(pool)
	authorRepo := repository.NewPostgresAuthorRepository(pool)
	logRepo := repository.NewPostgresLogRepository(pool)

	// 5. Services
	logService := service.NewLogServiceImpl(logRepo)
	ebookService := service.NewEBookServiceImpl(ebookRepo, logService)
	authorService := service.NewAuthorServiceImpl(authorRepo)

	// 6. Handlers
	ebookHandler := handlers.NewEBookHandler(ebookService)
	authorHandler := handlers.NewAuthorHandler(authorService)
	authHandler := handlers.NewAuthHandler(cfg.SupabaseURL, cfg.SupabaseAnonKey)
	webHandler := handlers.NewWebHandler(ebookService, logService, authorService)

	// 7. Router
	r := chi.NewRouter()
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(chimiddleware.Recoverer)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// MVC routes (public)
	r.Get("/", webHandler.ListCatalog)
	r.Get("/login", authHandler.ShowLogin)
	r.Post("/login", authHandler.Login)
	r.Get("/register", authHandler.ShowRegister)
	r.Post("/register", authHandler.Register)
	r.Post("/logout", authHandler.Logout)
	r.Get("/ebooks/{id}", webHandler.ShowEBook)
	r.Get("/authors", webHandler.ListAuthorsWeb)

	// MVC routes (protected)
	r.With(appmiddleware.RequireAuth(keySet)).
		Get("/ebooks/{id}/download", webHandler.DownloadEBook)

	// REST API routes (protegidas, acepta cookie o Bearer token)
	r.Route("/api/v1/ebooks", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuthAPI(keySet))
		ebookHandler.Routes(r)
	})
	r.Route("/api/v1/authors", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuthAPI(keySet))
		authorHandler.Routes(r)
	})
	r.With(appmiddleware.RequireAuthAPI(keySet)).
		Get("/api/v1/ebooks/{id}/author", authorHandler.GetAuthorByBookID)

	// HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 35 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("Server listening on :%s\n", cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
	}
}
