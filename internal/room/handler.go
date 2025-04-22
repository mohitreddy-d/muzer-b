package room

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/music-queue-system/pkg/models"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	rooms := r.Group("/rooms")
	{
		rooms.POST("/", h.createRoom)
		rooms.GET("/code/:code", h.getRoomByCode)
		rooms.GET("/:id", h.getRoom)
		rooms.POST("/:id/queue", h.addToQueue)
		rooms.GET("/:id/queue", h.getQueue)
		rooms.POST("/:id/vote", h.vote)
		rooms.GET("/:id/next", h.getNextSong)
	}
}

type CreateRoomRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *Handler) createRoom(c *gin.Context) {
	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id") // Set by auth middleware
	room, err := h.service.CreateRoom(c.Request.Context(), userID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, room)
}

func (h *Handler) getRoom(c *gin.Context) {
	roomID := c.Param("id")
	room, err := h.service.GetRoom(c.Request.Context(), roomID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, room)
}

func (h *Handler) getRoomByCode(c *gin.Context) {
	code := c.Param("code")
	room, err := h.service.GetRoomByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, room)
}

type AddToQueueRequest struct {
	TrackID   string `json:"track_id" binding:"required"`
	TrackName string `json:"track_name" binding:"required"`
	Artist    string `json:"artist" binding:"required"`
}

func (h *Handler) addToQueue(c *gin.Context) {
	roomID := c.Param("id")
	userID := c.GetString("user_id")

	var req AddToQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item := &models.QueueItem{
		RoomID:    uuid.MustParse(roomID),
		UserID:    uuid.MustParse(userID),
		TrackID:   req.TrackID,
		TrackName: req.TrackName,
		Artist:    req.Artist,
	}

	if err := h.service.AddToQueue(c.Request.Context(), roomID, item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

func (h *Handler) getQueue(c *gin.Context) {
	roomID := c.Param("id")
	queue, err := h.service.GetQueue(c.Request.Context(), roomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, queue)
}

type VoteRequest struct {
	TrackID string `json:"track_id" binding:"required"`
	Vote    int    `json:"vote" binding:"required,oneof=-1 1"`
}

func (h *Handler) vote(c *gin.Context) {
	roomID := c.Param("id")
	userID := c.GetString("user_id")

	var req VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.Vote(c.Request.Context(), roomID, req.TrackID, userID, req.Vote); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) getNextSong(c *gin.Context) {
	roomID := c.Param("id")
	song, err := h.service.GetNextSong(c.Request.Context(), roomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if song == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no songs in queue"})
		return
	}

	c.JSON(http.StatusOK, song)
}
