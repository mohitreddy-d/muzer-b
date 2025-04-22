package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/music-queue-system/internal/spotify"
	"github.com/music-queue-system/pkg/jwt"
	"github.com/music-queue-system/pkg/redis"
)

type Handler struct {
	spotifyClient *spotify.Client
	tokenStore    *redis.TokenStore
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func NewHandler(spotifyClient *spotify.Client, tokenStore *redis.TokenStore) *Handler {
	return &Handler{
		spotifyClient: spotifyClient,
		tokenStore:    tokenStore,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		// Public routes
		auth.GET("/login", h.login)

		auth.GET("/callback", h.callback)
		auth.GET("/refresh", h.refresh)

		// Protected routes (require authentication)
		protected := auth.Group("", AuthMiddleware(h.tokenStore))
		protected.GET("/user", h.User)
		protected.GET("/status", h.Status)
		protected.GET("/me/top-tracks", h.getTopTracks)
	}
}

func (h *Handler) Status(c *gin.Context) {
	token := c.GetString("access_token")
	fmt.Println(token, "access token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No access token"})
		return
	}
	// Check if token is valid
	user, err := h.spotifyClient.GetUser(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "user": user})
}

func (h *Handler) User(c *gin.Context) {
	user, err := h.spotifyClient.GetUser(c.Request.Context(), c.GetString("access_token"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) login(c *gin.Context) {
	state := uuid.New().String()
	authURL := h.spotifyClient.GetAuthURL(state)
	c.JSON(http.StatusOK, gin.H{"url": authURL})
}

func (h *Handler) callback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	// Exchange code for tokens
	token, err := h.spotifyClient.ExchangeToken(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate user ID if new user
	userID := uuid.New()

	// Store new token info
	tokenInfo := &redis.TokenInfo{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).UTC(),
	}

	if err := h.tokenStore.StoreTokens(c, userID.String(), tokenInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store tokens"})
		return
	}

	// Generate JWT
	jwtToken, err := jwt.GenerateToken(userID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Redirect back to frontend with token in cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "auth_token",
		Value:    jwtToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Set to true if using HTTPS
		SameSite: http.SameSiteStrictMode,
	})

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "/"
	}

	c.Redirect(http.StatusFound, frontendURL)
}

func (h *Handler) refresh(c *gin.Context) {
	userID := c.GetString("user_id")
	ctx := context.Background()

	// Get existing token
	tokenInfo, err := h.tokenStore.GetTokens(ctx, userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token not found"})
		return
	}

	// Refresh token with Spotify
	newToken, err := h.spotifyClient.RefreshToken(c.Request.Context(), tokenInfo.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update stored tokens
	if err := h.tokenStore.RefreshToken(c.Request.Context(), userID, newToken.AccessToken, time.Now().Add(time.Duration(newToken.ExpiresIn)*time.Second)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "token refreshed"})
}

func (h *Handler) getTopTracks(c *gin.Context) {
	accessToken := c.GetString("access_token")
	fmt.Println(accessToken, "access token")
	timeRange := c.DefaultQuery("time_range", "medium_term") // short_term, medium_term, long_term
	limit := 20

	tracks, err := h.spotifyClient.GetTopTracks(c.Request.Context(), accessToken, timeRange, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tracks)
}
