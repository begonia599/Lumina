package main

import (
	"log"
	"os"
	"strings"

	"lumina/internal/auth"
	"lumina/internal/database"
	"lumina/internal/handler"
	"lumina/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	// --- Database ---
	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// --- Upload directory ---
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	_ = os.MkdirAll(uploadDir, 0755)

	// --- Auth provider (ADR-13): swappable via interface ---
	registrationEnabled := envBool("REGISTRATION_ENABLED", true)
	secureCookie := envBool("SESSION_COOKIE_SECURE", false)
	var authProvider auth.Provider = auth.NewLocalProvider(database.Pool, auth.LocalProviderOptions{
		RegistrationEnabled: registrationEnabled,
	})
	localProvider := authProvider.(*auth.LocalAuthProvider)
	authHandler := handler.NewAuthHandler(
		authProvider,
		int(localProvider.SessionLifetime().Seconds()),
		secureCookie,
	)

	// --- Gin ---
	r := gin.Default()

	// CORS (dev-mode: Vite on 5173).
	// Credentials must be allowed so the session cookie is sent cross-origin.
	r.Use(cors.New(cors.Config{
		AllowOrigins:     devAllowedOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")

	// --- Public auth routes (rate-limited) ---
	api.POST("/auth/register", middleware.RateLimitPerMinute(3), authHandler.Register)
	api.POST("/auth/login", middleware.RateLimitPerMinute(5), authHandler.Login)

	// --- Authenticated routes ---
	authed := api.Group("")
	authed.Use(middleware.RequireAuth(authProvider))
	{
		authed.POST("/auth/logout", authHandler.Logout)
		authed.GET("/auth/me", authHandler.Me)

		// Books
		authed.POST("/books/upload", handler.UploadBook)
		authed.GET("/books", handler.ListBooks)
		authed.GET("/books/:id", handler.GetBook)
		authed.GET("/books/:id/resources/*path", handler.GetEPUBResource)
		authed.PATCH("/books/:id", handler.UpdateBook)
		authed.DELETE("/books/:id", handler.DeleteBook)

		// Book cover
		authed.POST("/books/:id/cover", handler.UploadCover)
		authed.DELETE("/books/:id/cover", handler.DeleteCover)
		authed.GET("/books/:id/cover", handler.GetCover)

		// Chapters
		authed.GET("/books/:id/chapters", handler.GetChapters)
		authed.GET("/books/:id/chapters/:idx", handler.GetChapterContent)

		// Search
		authed.GET("/books/:id/search", handler.SearchBook)

		// Reading progress
		authed.GET("/books/:id/progress", handler.GetProgress)
		authed.PUT("/books/:id/progress", handler.UpdateProgress)

		// Bookmarks
		authed.GET("/books/:id/bookmarks", handler.GetBookmarks)
		authed.POST("/books/:id/bookmarks", handler.CreateBookmark)
		authed.PATCH("/bookmarks/:id", handler.UpdateBookmark)
		authed.DELETE("/bookmarks/:id", handler.DeleteBookmark)

		// Settings
		authed.GET("/settings", handler.GetSettings)
		authed.PUT("/settings", handler.UpdateSettings)
	}

	// TODO (Phase 8): embed.FS for frontend dist + NoRoute fallback to index.html.

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("[Lumina] Server starting on :%s (registration=%v, secureCookie=%v)",
		port, registrationEnabled, secureCookie)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// envBool reads a boolean env var; returns fallback when unset.
func envBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "":
		return fallback
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// devAllowedOrigins parses CORS_ORIGINS (comma-separated) or falls back to
// the two common Vite / Next dev ports.
func devAllowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ORIGINS"))
	if raw == "" {
		return []string{"http://localhost:5173", "http://localhost:3000"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
