package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	goredis "github.com/redis/go-redis/v9"

	"github.com/music-queue-system/internal/auth"
	"github.com/music-queue-system/internal/room"
	"github.com/music-queue-system/internal/spotify"
	"github.com/music-queue-system/internal/ws"
	"github.com/music-queue-system/pkg/database"
	"github.com/music-queue-system/pkg/events"
	"github.com/music-queue-system/pkg/redis"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Set Gin mode based on environment
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize MySQL database
	db, err := database.NewMySQLDB(
		os.Getenv("MYSQL_HOST"),
		os.Getenv("MYSQL_PORT"),
		os.Getenv("MYSQL_USER"),
		os.Getenv("MYSQL_PASSWORD"),
		os.Getenv("MYSQL_DATABASE"),
	)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize Redis client
	redisClient := goredis.NewClient(&goredis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Initialize Kafka client
	kafkaClient := events.NewKafkaClient(
		strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
		"music-queue-events",
		os.Getenv("KAFKA_GROUP_ID"),
	)

	// Initialize services
	spotifyClient := spotify.NewClient(
		os.Getenv("SPOTIFY_CLIENT_ID"),
		os.Getenv("SPOTIFY_CLIENT_SECRET"),
		os.Getenv("SPOTIFY_REDIRECT_URI"),
	)

	tokenStore := redis.NewTokenStore(redisClient)
	roomService := room.NewService(db, redisClient, kafkaClient)

	// Initialize handlers
	authHandler := auth.NewHandler(spotifyClient, tokenStore)
	roomHandler := room.NewHandler(roomService)
	wsHandler := ws.NewHandler(kafkaClient)

	// Initialize Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "https://your-frontend-domain.com"}, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// API routes
	// Redirect legacy Spotify OAuth callback to the API route
	v1 := router.Group("/api/v1")
	router.GET("/auth/callback", func(c *gin.Context) {
		// Preserve query parameters when redirecting
		dest := "/api/v1/auth/callback"
		if raw := c.Request.URL.RawQuery; raw != "" {
			dest += "?" + raw
		}
		c.Redirect(http.StatusTemporaryRedirect, dest)
	})

	// Public routes
	authHandler.RegisterRoutes(v1)

	// Protected routes
	protected := v1.Group("/")
	protected.Use(auth.AuthMiddleware(tokenStore))
	{
		roomHandler.RegisterRoutes(protected)

		// WebSocket endpoint
		protected.GET("/ws/:roomId", wsHandler.HandleWebSocket)
	}
	// Serve frontend static files and SPA fallback
	router.NoRoute(func(c *gin.Context) {
		// Attempt to serve a static file
		// Construct path to dist
		reqPath := c.Request.URL.Path
		// Prevent directory traversal
		cleanPath := filepath.Clean(reqPath)
		filePath := filepath.Join("frontend/dist", cleanPath)
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			c.File(filePath)
		} else {
			// Fallback to index.html for client-side routing
			c.File("frontend/dist/index.html")
		}
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
