package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/music-queue-system/pkg/jwt"
	"github.com/music-queue-system/pkg/redis"
)

func AuthMiddleware(tokenStore *redis.TokenStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from header or query param (for WebSocket)
		//authHeader := c.GetHeader("Authorization")
		authHeader, _ := c.Cookie("auth_token")
		if authHeader == "" {
			if token := c.Query("token"); token != "" {
				authHeader = "Bearer " + token
			}
		}
		//fmt.Println(authHeader, "auth header")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No authorization header"})
			return
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		//fmt.Println(parts, "parts")
		//if len(parts) != 2 || parts[0] != "Bearer" {
		//	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
		//	return
		//}

		// Validate token
		claims, err := jwt.ValidateToken(parts[0])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Get token info from Redis
		tokenInfo, err := tokenStore.GetTokens(c.Request.Context(), claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token not found"})
			return
		}

		// Check if token is expired
		if time.Now().After(tokenInfo.ExpiresAt) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
			return
		}

		// Set user ID in context
		c.Set("user_id", claims.UserID)
		c.Set("access_token", tokenInfo.AccessToken)
		//fmt.Println(tokenInfo.AccessToken, "access token*1")
		c.Next()
	}
}
