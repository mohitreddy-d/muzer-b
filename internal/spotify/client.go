package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	clientID     string
	clientSecret string
	redirectURI  string
	httpClient   *http.Client
}
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	ExpiresAt    time.Time
}

func (tr *TokenResponse) addExpiresAt() {
	tr.ExpiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
}

type Track struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Artists  []Artist `json:"artists"`
	Duration int      `json:"duration_ms"`
	Album    Album    `json:"album"`
}

type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Album struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Images []Image `json:"images"`
}

type Image struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

type SearchResponse struct {
	Tracks struct {
		Items []Track `json:"items"`
	} `json:"tracks"`
}

type TopTracksResponse struct {
	Items []Track `json:"items"`
}

func NewClient(clientID, clientSecret, redirectURI string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetAuthURL(state string) string {
	params := url.Values{}
	params.Add("client_id", c.clientID)
	params.Add("response_type", "code")
	params.Add("redirect_uri", c.redirectURI)
	params.Add("scope", "user-read-private user-read-email playlist-read-private user-top-read streaming user-read-playback-state user-modify-playback-state")
	params.Add("state", state)

	return "https://accounts.spotify.com/authorize?" + params.Encode()
}

func (c *Client) ExchangeToken(ctx context.Context, code string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", c.redirectURI)

	return c.doTokenRequest(ctx, data)
}

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	return c.doTokenRequest(ctx, data)
}

func (c *Client) doTokenRequest(ctx context.Context, data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify: token request failed with status %d", resp.StatusCode)
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	fmt.Println("expires in", token.ExpiresIn, "expires at:", token.ExpiresAt)
	return &token, nil
}

func (c *Client) SearchTracks(ctx context.Context, accessToken, query string, limit int) ([]Track, error) {
	params := url.Values{}
	params.Add("q", query)
	params.Add("type", "track")
	params.Add("limit", fmt.Sprintf("%d", limit))

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.spotify.com/v1/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify: search request failed with status %d", resp.StatusCode)
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return searchResp.Tracks.Items, nil
}

func (c *Client) GetTopTracks(ctx context.Context, accessToken string, timeRange string, limit int) ([]Track, error) {
	params := url.Values{}
	params.Add("time_range", timeRange) // short_term, medium_term, long_term
	params.Add("limit", fmt.Sprintf("%d", limit))

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.spotify.com/v1/me/top/tracks?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify: top tracks request failed with status %d", resp.StatusCode)
	}

	var topTracksResp TopTracksResponse
	if err := json.NewDecoder(resp.Body).Decode(&topTracksResp); err != nil {
		return nil, err
	}

	return topTracksResp.Items, nil
}

func (c *Client) PlayTrack(ctx context.Context, accessToken, deviceID, trackURI string) error {
	payload := map[string]interface{}{
		"uris": []string{fmt.Sprintf("spotify:track:%s", trackURI)},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := "https://api.spotify.com/v1/me/player/play"
	if deviceID != "" {
		url += "?device_id=" + deviceID
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("spotify: play track request failed with status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) GetUser(ctx context.Context, accessToken string) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.spotify.com/v1/me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify: get user request failed with status %d", resp.StatusCode)
	}

	return resp.Body, nil
}
