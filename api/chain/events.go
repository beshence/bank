package chain

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"bank/internal/app"
	"bank/internal/database/models"
	"bank/internal/middleware"

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

		accountID, ok := middleware.GetCurrentAccount(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		accountUUID, err := uuid.Parse(accountID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		vaultID, err := uuid.Parse(c.Param("vaultId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid bank id"})
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

		chainID, err := uuid.Parse(c.Param("chainId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid chain id"})
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

		if _, err := loadVaultForAccount(tx, vaultID, accountUUID); err != nil {
			rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "bank not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load bank"})
			return
		}

		var chain models.Chain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND vault_id = ?", chainID, vaultID).Take(&chain).Error; err != nil {
			rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "chain not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load chain"})
			return
		}

		if !sameEventID(parentID, chain.LastEventID) {
			rollback()
			c.JSON(http.StatusConflict, conflictResponse{
				Error:             "conflict",
				ServerLastEventID: uuidToStringPtr(chain.LastEventID),
			})
			return
		}

		event := models.Event{
			EventID:   eventID,
			ChainID:   chainID,
			ParentID:  parentID,
			SessionID: sessionID,
			Payload:   request.Payload,
		}

		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "event_id"}}, DoNothing: true}).Create(&event)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				rollback()
				c.JSON(http.StatusConflict, conflictResponse{
					Error:             "conflict",
					ServerLastEventID: uuidToStringPtr(chain.LastEventID),
				})
				return
			}

			rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to insert event"})
			return
		}

		if result.RowsAffected == 0 {
			var existing models.Event
			if err := tx.Where("chain_id = ? AND event_id = ?", chainID, eventID).Take(&existing).Error; err != nil {
				rollback()
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusConflict, conflictResponse{
						Error:             "conflict",
						ServerLastEventID: uuidToStringPtr(chain.LastEventID),
					})
					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to check existing event"})
				return
			}

			if !sameEventID(existing.ParentID, parentID) || existing.Payload != request.Payload {
				rollback()
				c.JSON(http.StatusConflict, conflictResponse{
					Error:             "conflict",
					ServerLastEventID: uuidToStringPtr(chain.LastEventID),
				})
				return
			}

			if err := tx.Commit().Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to complete transaction"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		}

		if err := tx.Model(&models.Chain{}).Where("id = ?", chainID).Update("last_event_id", eventID).Error; err != nil {
			rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update chain state"})
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

		accountID, ok := middleware.GetCurrentAccount(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		accountUUID, err := uuid.Parse(accountID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		vaultID, err := uuid.Parse(c.Param("vaultId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid bank id"})
			return
		}

		chainID, err := uuid.Parse(c.Param("chainId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid chain id"})
			return
		}

		if _, err := loadVaultForAccount(deps.DB, vaultID, accountUUID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "bank not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load bank"})
			return
		}

		var chain models.Chain
		if err := deps.DB.Where("id = ? AND vault_id = ?", chainID, vaultID).Take(&chain).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "chain not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load chain"})
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

		events, err := fetchEventsAfterCursor(deps.DB, chainID, sinceID, limit)
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
		return nil, fmt.Errorf("invalid cursor")
	}

	return &parsed, nil
}

func fetchEventsAfterCursor(db *gorm.DB, chainID uuid.UUID, sinceID *uuid.UUID, limit int) ([]models.Event, error) {
	if limit <= 0 {
		return []models.Event{}, nil
	}

	if sinceID != nil {
		var base models.Event
		if err := db.Where("chain_id = ? AND event_id = ?", chainID, *sinceID).Take(&base).Error; err != nil {
			return nil, err
		}
	}

	tableName, err := eventTableName(db)
	if err != nil {
		return nil, err
	}

	events := make([]models.Event, 0, limit)

	if sinceID == nil {
		query := fmt.Sprintf(`
WITH RECURSIVE event_chain AS (
    SELECT e.*
    FROM %s AS e
    WHERE e.chain_id = ? AND e.parent_id IS NULL
    LIMIT 1

    UNION ALL

    SELECT child.*
    FROM event_chain AS ec
    JOIN LATERAL (
        SELECT e.*
        FROM %s AS e
        WHERE e.chain_id = ? AND e.parent_id = ec.event_id
        LIMIT 1
    ) AS child ON true
)
SELECT *
FROM event_chain
LIMIT ?`, tableName, tableName)

		if err := db.Raw(query, chainID, chainID, limit).Scan(&events).Error; err != nil {
			return nil, err
		}

		return events, nil
	}

	query := fmt.Sprintf(`
WITH RECURSIVE event_chain AS (
    SELECT e.*
    FROM %s AS e
    WHERE e.chain_id = ? AND e.event_id = ?
    LIMIT 1

    UNION ALL

    SELECT child.*
    FROM event_chain AS ec
    JOIN LATERAL (
        SELECT e.*
        FROM %s AS e
        WHERE e.chain_id = ? AND e.parent_id = ec.event_id
        LIMIT 1
    ) AS child ON true
)
SELECT *
FROM event_chain
WHERE event_id <> ?
LIMIT ?`, tableName, tableName)

	if err := db.Raw(query, chainID, *sinceID, chainID, *sinceID, limit).Scan(&events).Error; err != nil {
		return nil, err
	}

	return events, nil
}

func eventTableName(db *gorm.DB) (string, error) {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(&models.Event{}); err != nil {
		return "", err
	}

	return stmt.Schema.Table, nil
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
