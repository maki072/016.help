package main

import (
	"context"
	"fmt"
	"helpdesk/internal/auth"
	"helpdesk/internal/bot"
	"helpdesk/internal/calendar"
	"helpdesk/internal/db"
	"helpdesk/internal/handlers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Load environment variables
	if err := loadEnv(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Initialize database
	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		log.Printf("Warning: Migration error (may be normal if tables exist): %v", err)
	}

	// Create uploads directory
	if err := os.MkdirAll("uploads", 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}

	// Initialize templates
	if err := handlers.InitTemplates(); err != nil {
		log.Fatalf("Failed to initialize templates: %v", err)
	}

	// Initialize Google Calendar
	calendar.Init()

	// Initialize Telegram bot
	if err := bot.Init(); err != nil {
		log.Printf("Warning: Failed to initialize Telegram bot: %v", err)
		log.Println("Continuing without Telegram bot...")
	} else {
		// Start bot in goroutine
		go func() {
			log.Println("Starting Telegram bot...")
			bot.Start()
		}()
	}

	// Setup HTTP router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Public routes
	r.Get("/login", handlers.LoginHandler)
	r.Post("/login", handlers.LoginHandler)
	r.Get("/logout", handlers.LogoutHandler)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cookie, err := r.Cookie("session")
				if err != nil {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}

				session, err := auth.GetSession(cookie.Value)
				if err != nil {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}

				// Store session in headers (simplified - in production use context)
				r.Header.Set("X-User-ID", fmt.Sprintf("%d", session.UserID))
				r.Header.Set("X-Organization-ID", fmt.Sprintf("%d", session.OrganizationID))
				r.Header.Set("X-User-Role", session.Role)

				next.ServeHTTP(w, r)
			})
		})
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		})
		r.Get("/dashboard", handlers.DashboardHandler)
		r.Get("/ticket/{id}", handlers.TicketHandler)
		r.Post("/ticket/message", handlers.AddMessageHandler)
		r.Post("/ticket/status", handlers.UpdateTicketStatusHandler)
		r.Post("/ticket/assign", handlers.AssignTicketHandler)
		r.Get("/auth/google", handlers.GoogleCalendarAuthHandler)
		r.Get("/auth/google/callback", handlers.GoogleCalendarCallbackHandler)
	})

	// Start HTTP server
	httpPort := getEnv("HTTP_PORT", "8080")
	httpHost := getEnv("HTTP_HOST", "0.0.0.0")
	addr := fmt.Sprintf("%s:%s", httpHost, httpPort)

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting HTTP server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func loadEnv() error {
	// Simple .env loader (in production, use godotenv or similar)
	// For now, we rely on environment variables being set
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
