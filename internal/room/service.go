package room

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/music-queue-system/pkg/database"
	"github.com/music-queue-system/pkg/events"
	"github.com/music-queue-system/pkg/models"
)

const (
	roomKeyPrefix    = "room:"
	roomQueuePrefix = "queue:"
	codeLength      = 6
)

type Service struct {
	db      *database.MySQLDB
	redis   *redis.Client
	events  *events.KafkaClient
}

func NewService(db *database.MySQLDB, redis *redis.Client, events *events.KafkaClient) *Service {
	return &Service{
		db:     db,
		redis:  redis,
		events: events,
	}
}

func (s *Service) CreateRoom(ctx context.Context, hostID string, name string) (*models.Room, error) {
	room := &models.Room{
		ID:        uuid.New(),
		Code:      generateRoomCode(),
		HostID:    uuid.MustParse(hostID),
		Name:      name,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store room in MySQL
	if err := s.db.CreateRoom(room); err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	// Cache room in Redis
	roomJSON, err := json.Marshal(room)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal room: %w", err)
	}

	key := fmt.Sprintf("%s%s", roomKeyPrefix, room.ID)
	if err := s.redis.Set(ctx, key, roomJSON, 24*time.Hour).Err(); err != nil {
		log.Printf("Warning: failed to cache room: %v", err)
	}

	return room, nil
}

func (s *Service) GetRoom(ctx context.Context, roomID string) (*models.Room, error) {
	// Try cache first
	key := fmt.Sprintf("%s%s", roomKeyPrefix, roomID)
	roomJSON, err := s.redis.Get(ctx, key).Bytes()
	if err == nil {
		var room models.Room
		if err := json.Unmarshal(roomJSON, &room); err == nil {
			return &room, nil
		}
	}

	// Fallback to database
	room, err := s.db.GetRoomByID(roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	// Update cache
	roomJSON, _ = json.Marshal(room)
	s.redis.Set(ctx, key, roomJSON, 24*time.Hour)

	return room, nil
}

func (s *Service) GetRoomByCode(ctx context.Context, code string) (*models.Room, error) {
	// Try getting from database
	room, err := s.db.GetRoomByCode(code)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	// Cache the result
	roomJSON, _ := json.Marshal(room)
	key := fmt.Sprintf("%s%s", roomKeyPrefix, room.ID)
	s.redis.Set(ctx, key, roomJSON, 24*time.Hour)

	return room, nil
}

func (s *Service) AddToQueue(ctx context.Context, roomID string, item *models.QueueItem) error {
	// Get current queue from database
	queue, err := s.db.GetQueue(roomID)
	if err != nil {
		return fmt.Errorf("failed to get queue: %w", err)
	}
	
	item.Position = len(queue)
	item.ID = uuid.New()
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()

	// Add to database
	if err := s.db.AddToQueue(item); err != nil {
		return fmt.Errorf("failed to add to queue: %w", err)
	}

	// Publish event
	payload := events.SongAddedPayload{
		TrackID:   item.TrackID,
		TrackName: item.TrackName,
		Artist:    item.Artist,
	}

	if err := s.events.PublishEvent(ctx, "song-additions", payload); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

func (s *Service) GetQueue(ctx context.Context, roomID string) ([]*models.QueueItem, error) {
	// Get queue from database
	queue, err := s.db.GetQueue(roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}

	return queue, nil
}

func (s *Service) Vote(ctx context.Context, roomID string, trackID string, userID string, voteValue int) error {
	vote := &models.Vote{
		ID:          uuid.New(),
		QueueItemID: uuid.MustParse(trackID),
		UserID:      uuid.MustParse(userID),
		Value:       voteValue,
		CreatedAt:   time.Now(),
	}

	// Store vote in database
	if err := s.db.CreateOrUpdateVote(vote); err != nil {
		return fmt.Errorf("failed to store vote: %w", err)
	}

	// Get updated vote count and publish event
	total, err := s.db.GetVotesForItem(trackID)
	if err != nil {
		return fmt.Errorf("failed to get total votes: %w", err)
	}

	// Publish vote event with total
	s.events.PublishVoteUpdate(ctx, roomID, trackID, total)

	// Publish vote event
	payload := events.SongVotedPayload{
		TrackID: trackID,
		UserID:  userID,
		Value:   voteValue,
	}

	if err := s.events.PublishEvent(ctx, "song-votes", payload); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

func (s *Service) calculateTotalVotes(ctx context.Context, voteKey string) (int, error) {
	votes, err := s.redis.HGetAll(ctx, voteKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get votes: %w", err)
	}

	total := 0
	for _, vote := range votes {
		if vote == "1" {
			total++
		} else if vote == "-1" {
			total--
		}
	}

	return total, nil
}

func generateRoomCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, codeLength)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

func (s *Service) GetNextSong(ctx context.Context, roomID string) (*models.QueueItem, error) {
	// Get next song directly from database (ordered by votes)
	nextSong, err := s.db.GetNextSong(roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next song: %w", err)
	}

	return nextSong, nil
}
