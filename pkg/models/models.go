package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `json:"id" gorm:"primaryKey"`
	SpotifyID   string    `json:"spotify_id" gorm:"unique"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Room struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey"`
	Code      string    `json:"code" gorm:"unique"`
	HostID    uuid.UUID `json:"host_id"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type QueueItem struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey"`
	RoomID    uuid.UUID `json:"room_id"`
	UserID    uuid.UUID `json:"user_id"`
	TrackID   string    `json:"track_id"`
	TrackName string    `json:"track_name"`
	Artist    string    `json:"artist"`
	Votes     int       `json:"votes"`
	Position  int       `json:"position"`
	Played    bool      `json:"played"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Vote struct {
	ID          uuid.UUID `json:"id" gorm:"primaryKey"`
	QueueItemID uuid.UUID `json:"queue_item_id"`
	UserID      uuid.UUID `json:"user_id"`
	Value       int       `json:"value"` // 1 for upvote, -1 for downvote
	CreatedAt   time.Time `json:"created_at"`
}
