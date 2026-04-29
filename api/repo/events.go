package repo

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"vault/internal/app"
	"vault/internal/database/models"
	"vault/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SyncRequest struct {
	Since       string `form:"since"`
	LastEventID string `json:"last_event_id" form:"last_event_id"`
}

type SyncResponse struct {
	Events []models.Event `json:"events"`
}

type appendEventRequest struct {
	EventID  string  `json:"event_id" binding:"required"`
	ParentID *string `json:"parent_id"`
	Payload  string  `json:"payload" binding:"required"`
}

type conflictResponse struct {
	Error             string  `json:"error"`
	ServerLastEventID *string `json:"server_last_event_id"`
}

func AppendEventV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "database is not configured"})
			return
		}

		if _, ok := middleware.GetCurrentUser(c); !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		sessionIDRaw, _, sessionOK := middleware.GetCurrentSession(c)
		if !sessionOK {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		sessionID, err := uuid.Parse(sessionIDRaw)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		repoID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid repository id"})
			return
		}

		var request appendEventRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		eventID, err := uuid.Parse(request.EventID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid event_id"})
			return
		}

		parentID, err := parseOptionalUUID(request.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid parent_id"})
			return
		}

		tx := deps.DB.Begin()
		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to start transaction"})
			return
		}

		rollback := func() {
			_ = tx.Rollback().Error
		}

		var existing models.Event
		if err := tx.Where("event_id = ?", eventID).Take(&existing).Error; err == nil {
			if err := tx.Commit().Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to complete transaction"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to check event id"})
			return
		}

		var repository models.Repository
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", repoID).Take(&repository).Error; err != nil {
			rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "repository not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load repository"})
			return
		}

		if !sameEventID(parentID, repository.LastEventID) {
			rollback()
			c.JSON(http.StatusConflict, conflictResponse{
				Error:             "conflict",
				ServerLastEventID: uuidToStringPtr(repository.LastEventID),
			})
			return
		}

		event := models.Event{
			EventID:      eventID,
			RepositoryID: repoID,
			ParentID:     parentID,
			SessionID:    sessionID,
			Payload:      request.Payload,
		}

		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "event_id"}}, DoNothing: true}).Create(&event)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				rollback()
				c.JSON(http.StatusConflict, conflictResponse{
					Error:             "conflict",
					ServerLastEventID: uuidToStringPtr(repository.LastEventID),
				})
				return
			}

			rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to insert event"})
			return
		}

		if result.RowsAffected == 0 {
			if err := tx.Commit().Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to complete transaction"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		}

		if err := tx.Model(&models.Repository{}).Where("id = ?", repoID).Update("last_event_id", eventID).Error; err != nil {
			rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update repository state"})
			return
		}

		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to complete transaction"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"status": "ok"})
	}
}

func FetchEventsV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "database is not configured"})
			return
		}

		if _, ok := middleware.GetCurrentUser(c); !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		repoID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid repository id"})
			return
		}

		var repository models.Repository
		if err := deps.DB.Where("id = ?", repoID).Take(&repository).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "repository not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load repository"})
			return
		}

		var request SyncRequest
		_ = c.ShouldBindQuery(&request)

		sinceID, err := parseFetchCursor(request)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		limit := 100
		if requestLimit := c.Query("limit"); requestLimit != "" {
			parsedLimit, parseErr := strconv.Atoi(requestLimit)
			if parseErr != nil || parsedLimit <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"message": "invalid limit"})
				return
			}
			if parsedLimit < limit {
				limit = parsedLimit
			}
		}

		events, err := fetchEventsAfterCursor(deps.DB, repoID, sinceID, limit)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "since event not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load events"})
			return
		}

		c.JSON(http.StatusOK, SyncResponse{Events: events})
	}
}

func parseOptionalUUID(value *string) (*uuid.UUID, error) {
	if value == nil || *value == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(*value)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func parseFetchCursor(request SyncRequest) (*uuid.UUID, error) {
	rawCursor := request.Since
	if rawCursor == "" {
		rawCursor = request.LastEventID
	}

	if rawCursor == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(rawCursor)
	if err != nil {
		return nil, fmt.Errorf("invalid last_event_id")
	}

	return &parsed, nil
}

func fetchEventsAfterCursor(db *gorm.DB, repoID uuid.UUID, sinceID *uuid.UUID, limit int) ([]models.Event, error) {
	events := make([]models.Event, 0, limit)
	current := sinceID

	if current != nil {
		var base models.Event
		if err := db.Where("repository_id = ? AND event_id = ?", repoID, *current).Take(&base).Error; err != nil {
			return nil, err
		}
	}

	for len(events) < limit {
		next, err := loadNextEvent(db, repoID, current)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}

			return nil, err
		}

		events = append(events, *next)
		current = &next.EventID
	}

	return events, nil
}

func loadNextEvent(db *gorm.DB, repoID uuid.UUID, parentID *uuid.UUID) (*models.Event, error) {
	query := db.Where("repository_id = ?", repoID)
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}

	var event models.Event
	if err := query.Take(&event).Error; err != nil {
		return nil, err
	}

	return &event, nil
}

func sameEventID(a, b *uuid.UUID) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}

func uuidToStringPtr(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}

	text := value.String()
	return &text
}
