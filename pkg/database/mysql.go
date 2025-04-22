package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/music-queue-system/pkg/models"
)

type MySQLDB struct {
	*gorm.DB
}

func NewMySQLDB(host, port, user, password, dbname string) (*MySQLDB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbname)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto-migrate the schema
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &MySQLDB{DB: db}, nil
}

func autoMigrate(db *gorm.DB) error {
	log.Println("Running database migrations...")
	
	return db.AutoMigrate(
		&models.User{},
		&models.Room{},
		&models.QueueItem{},
		&models.Vote{},
	)
}

// User operations
func (db *MySQLDB) CreateUser(user *models.User) error {
	return db.Create(user).Error
}

func (db *MySQLDB) GetUserByID(id string) (*models.User, error) {
	var user models.User
	if err := db.First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (db *MySQLDB) GetUserBySpotifyID(spotifyID string) (*models.User, error) {
	var user models.User
	if err := db.First(&user, "spotify_id = ?", spotifyID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Room operations
func (db *MySQLDB) CreateRoom(room *models.Room) error {
	return db.Create(room).Error
}

func (db *MySQLDB) GetRoomByID(id string) (*models.Room, error) {
	var room models.Room
	if err := db.First(&room, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (db *MySQLDB) GetRoomByCode(code string) (*models.Room, error) {
	var room models.Room
	if err := db.First(&room, "code = ?", code).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (db *MySQLDB) UpdateRoom(room *models.Room) error {
	return db.Save(room).Error
}

// Queue operations
func (db *MySQLDB) AddToQueue(item *models.QueueItem) error {
	return db.Create(item).Error
}

func (db *MySQLDB) GetQueue(roomID string) ([]*models.QueueItem, error) {
	var items []*models.QueueItem
	if err := db.Where("room_id = ? AND played = ?", roomID, false).
		Order("votes DESC, created_at ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (db *MySQLDB) UpdateQueueItem(item *models.QueueItem) error {
	return db.Save(item).Error
}

// Vote operations
func (db *MySQLDB) CreateOrUpdateVote(vote *models.Vote) error {
	var existing models.Vote
	result := db.Where("queue_item_id = ? AND user_id = ?", vote.QueueItemID, vote.UserID).First(&existing)
	
	if result.Error == gorm.ErrRecordNotFound {
		return db.Create(vote).Error
	}

	existing.Value = vote.Value
	return db.Save(&existing).Error
}

func (db *MySQLDB) GetVotesForItem(queueItemID string) (int, error) {
	var sum struct {
		Total int
	}
	
	if err := db.Model(&models.Vote{}).
		Select("COALESCE(SUM(value), 0) as total").
		Where("queue_item_id = ?", queueItemID).
		Scan(&sum).Error; err != nil {
		return 0, err
	}

	return sum.Total, nil
}

func (db *MySQLDB) GetNextSong(roomID string) (*models.QueueItem, error) {
	var item models.QueueItem
	if err := db.Where("room_id = ? AND played = ?", roomID, false).
		Order("votes DESC, created_at ASC").
		First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
