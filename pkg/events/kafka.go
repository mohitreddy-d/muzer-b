package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type EventType string

const (
	EventTypeSongAdded     EventType = "song_added"
	EventTypeSongVoted     EventType = "song_voted"
	EventTypeSongStarted   EventType = "song_started"
	EventTypeSongCompleted EventType = "song_completed"
	EventTypeUserJoined    EventType = "user_joined"
	EventTypeUserLeft      EventType = "user_left"
)

type Event struct {
	Type      EventType         `json:"type"`
	RoomID    string           `json:"room_id"`
	UserID    string           `json:"user_id"`
	Timestamp time.Time        `json:"timestamp"`
	Payload   json.RawMessage  `json:"payload"`
}

type KafkaClient struct {
	writer *kafka.Writer
	reader *kafka.Reader
}

func NewKafkaClient(brokers []string, topic string, groupID string) *KafkaClient {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: kafka.LastOffset,
	})

	return &KafkaClient{
		writer: writer,
		reader: reader,
	}
}

func (k *KafkaClient) PublishEvent(ctx context.Context, topic string, payload interface{}) error {
	messageJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(uuid.New().String()),
		Value: messageJSON,
	}

	if err := k.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

func (k *KafkaClient) PublishVoteUpdate(ctx context.Context, roomID, trackID string, totalVotes int) error {
	payload := VoteUpdatePayload{
		RoomID:     roomID,
		TrackID:    trackID,
		TotalVotes: totalVotes,
		Timestamp:  time.Now(),
	}

	return k.PublishEvent(ctx, "vote-updates", payload)
}

func (k *KafkaClient) ConsumeEvents(ctx context.Context, handler func(Event) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := k.reader.ReadMessage(ctx)
			if err != nil {
				return fmt.Errorf("failed to read message: %w", err)
			}

			var event Event
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				return fmt.Errorf("failed to unmarshal event: %w", err)
			}

			if err := handler(event); err != nil {
				return fmt.Errorf("failed to handle event: %w", err)
			}
		}
	}
}

func (k *KafkaClient) Close() error {
	if err := k.writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}
	if err := k.reader.Close(); err != nil {
		return fmt.Errorf("failed to close reader: %w", err)
	}
	return nil
}

// Event payload types
type SongAddedPayload struct {
	TrackID   string `json:"track_id"`
	TrackName string `json:"track_name"`
	Artist    string `json:"artist"`
}

type SongVotedPayload struct {
	TrackID string `json:"track_id"`
	UserID  string `json:"user_id"`
	Value   int    `json:"value"`
}

type VoteUpdatePayload struct {
	RoomID     string    `json:"room_id"`
	TrackID    string    `json:"track_id"`
	TotalVotes int       `json:"total_votes"`
	Timestamp  time.Time `json:"timestamp"`
}

type SongStartedPayload struct {
	TrackID   string `json:"track_id"`
	TrackName string `json:"track_name"`
	Artist    string `json:"artist"`
}

type UserJoinedPayload struct {
	UserName string `json:"user_name"`
}
