package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenInfo struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type TokenStore struct {
	client *redis.Client
}

// NewTokenStore creates a new token store with the given Redis client
func NewTokenStore(client *redis.Client) *TokenStore {
	return &TokenStore{client: client}
}

// StoreTokens stores the user's Spotify tokens in Redis
func (s *TokenStore) StoreTokens(ctx context.Context, userID string, token *TokenInfo) error {
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	key := fmt.Sprintf("token:%s", userID)
	if err := s.client.Set(ctx, key, tokenJSON, 0).Err(); err != nil { // 0 means no expiration
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

// GetTokens retrieves the user's Spotify tokens from Redis
func (s *TokenStore) GetTokens(ctx context.Context, userID string) (*TokenInfo, error) {
	key := fmt.Sprintf("token:%s", userID)
	tokenJSON, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var token TokenInfo
	if err := json.Unmarshal(tokenJSON, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// DeleteToken removes the user's tokens from Redis
func (s *TokenStore) DeleteToken(ctx context.Context, userID string) error {
	key := fmt.Sprintf("token:%s", userID)
	return s.client.Del(ctx, key).Err()
}

// RefreshToken updates the access token and its expiry in Redis
func (s *TokenStore) RefreshToken(ctx context.Context, userID string, newAccessToken string, newExpiresAt time.Time) error {
	token, err := s.GetTokens(ctx, userID)
	if err != nil {
		return err
	}

	token.AccessToken = newAccessToken
	token.ExpiresAt = newExpiresAt
	return s.StoreTokens(ctx, userID, token)
}
