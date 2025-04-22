package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/music-queue-system/pkg/events"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, implement proper origin checking
	},
}

type VoteMessage struct {
	TrackID string `json:"track_id"`
	Value   int    `json:"value"`
}

type SongStartMessage struct {
	TrackID   string `json:"track_id"`
	TrackName string `json:"track_name"`
}

type Handler struct {
	// Map of roomID -> map of connectionID -> *websocket.Conn
	rooms    map[string]map[string]*websocket.Conn
	mu       sync.RWMutex
	events   *events.KafkaClient
}

func NewHandler(events *events.KafkaClient) *Handler {
	return &Handler{
		rooms:  make(map[string]map[string]*websocket.Conn),
		events: events,
	}
}

func (h *Handler) HandleWebSocket(c *gin.Context) {
	roomID := c.Param("roomId")
	if roomID == "" {
		c.JSON(400, gin.H{"error": "room_id is required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	connID := c.GetString("user_id") // Set by auth middleware
	h.addConnection(roomID, connID, conn)
	defer h.removeConnection(roomID, connID)

	// Start consuming Kafka events for this room
	go h.consumeEvents(roomID)

	// Handle incoming messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// Handle different message types
		switch msg["type"] {
		case "vote":
			var voteMsg VoteMessage
			if err := json.Unmarshal(message, &voteMsg); err != nil {
				log.Printf("Failed to parse vote message: %v", err)
				continue
			}
			h.handleVote(roomID, connID, voteMsg)
		case "add_song":
			h.handleAddSong(roomID, connID, msg)
		case "song_start":
			var songMsg SongStartMessage
			if err := json.Unmarshal(message, &songMsg); err != nil {
				log.Printf("Failed to parse song start message: %v", err)
				continue
			}
			h.handleSongStart(roomID, songMsg)
		}
	}
}

func (h *Handler) addConnection(roomID, connID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.rooms[roomID]; !exists {
		h.rooms[roomID] = make(map[string]*websocket.Conn)
	}
	h.rooms[roomID][connID] = conn

	// Notify others that a user has joined
	h.broadcastToRoom(roomID, map[string]interface{}{
		"type":    "user_joined",
		"user_id": connID,
	})
}

func (h *Handler) removeConnection(roomID, connID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, exists := h.rooms[roomID]; exists {
		if conn, exists := room[connID]; exists {
			conn.Close()
			delete(room, connID)
		}
		if len(room) == 0 {
			delete(h.rooms, roomID)
		}
	}

	// Notify others that a user has left
	h.broadcastToRoom(roomID, map[string]interface{}{
		"type":    "user_left",
		"user_id": connID,
	})
}

func (h *Handler) broadcastToRoom(roomID string, message interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if room, exists := h.rooms[roomID]; exists {
		messageJSON, err := json.Marshal(message)
		if err != nil {
			log.Printf("Failed to marshal message: %v", err)
			return
		}

		for _, conn := range room {
			if err := conn.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
				log.Printf("Failed to send message: %v", err)
			}
		}
	}
}

func (h *Handler) consumeEvents(roomID string) {
	ctx := context.Background()
	err := h.events.ConsumeEvents(ctx, func(event events.Event) error {
		if event.RoomID == roomID {
			h.broadcastToRoom(roomID, event)
		}
		return nil
	})
	if err != nil {
		log.Printf("Failed to consume events: %v", err)
	}
}

func (h *Handler) handleVote(roomID, userID string, msg VoteMessage) {
	// Publish vote event
	payload := events.SongVotedPayload{
		TrackID: msg.TrackID,
		UserID:  userID,
		Value:   msg.Value,
	}

	if err := h.events.PublishEvent(context.Background(), "song-votes", payload); err != nil {
		log.Printf("Failed to publish vote event: %v", err)
	}
}

func (h *Handler) handleSongStart(roomID string, msg SongStartMessage) {
	// Publish song start event
	payload := events.SongStartedPayload{
		TrackID:   msg.TrackID,
		TrackName: msg.TrackName,
	}

	if err := h.events.PublishEvent(context.Background(), "song-starts", payload); err != nil {
		log.Printf("Failed to publish song start event: %v", err)
	}
}

func (h *Handler) handleAddSong(roomID, userID string, msg map[string]interface{}) {
	trackID, ok := msg["track_id"].(string)
	if !ok {
		return
	}

	trackName, ok := msg["track_name"].(string)
	if !ok {
		return
	}

	artist, ok := msg["artist"].(string)
	if !ok {
		return
	}

	// Publish song added event
	payload := events.SongAddedPayload{
		TrackID:   trackID,
		TrackName: trackName,
		Artist:    artist,
	}

	if err := h.events.PublishEvent(context.Background(), "song-additions", payload); err != nil {
		log.Printf("Failed to publish song added event: %v", err)
	}
}
