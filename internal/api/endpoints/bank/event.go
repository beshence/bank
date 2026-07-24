package bank

import (
	"bank/internal/api"
	"bank/internal/auth"
	"bank/internal/database/models"
	"bank/internal/misc"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type addEventRequest struct {
	EventID  string  `json:"id" binding:"required"`
	ParentID *string `json:"parent_id"`
	Payload  string  `json:"payload" binding:"required"`
}

type eventsRequest struct {
	AfterID string `json:"after" form:"after"`
}

type eventsResponse struct {
	Events []models.Event `json:"events"`
	Err    string         `json:"err"`
}

type lastEventResponse struct {
	Event *models.Event `json:"event"`
	Err   string        `json:"err"`
}

// TODO: redo all ooops

func EventsV1(deps *api.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "database is not configured",
			})
			return
		}

		chainName := c.Param("chainName")

		// TODO: check for name

		tokenType, ok := auth.GetCurrentTokenType(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "unauthorized",
			})
			return
		}

		var accountID uuid.UUID
		var vaultID uuid.UUID

		if tokenType == auth.TokenTypeOauth {
			var scope string
			accountID, vaultID, scope, ok = auth.GetCurrentOauthContext(c, deps.DB)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "unauthorized",
				})
				return
			}

			parts := strings.Split(scope, ";")
			found := slices.Contains(parts, "chain:"+chainName+":r") || slices.Contains(parts, "chain:"+chainName+":rw")

			if !found {
				c.JSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "unauthorized",
				})
			}
		} else {
			accountID, ok = auth.GetCurrentAccount(c)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "unauthorized",
				})
				return
			}

			var err error
			vaultID, err = uuid.Parse(c.Param("vaultId"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "invalid vault id",
				})
				return
			}
		}

		if _, err := loadVaultForAccount(deps.DB, vaultID, accountID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "vault not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to load vault",
			})
			return
		}

		var chain models.Chain
		if err := deps.DB.Where("name = ? AND vault_id = ?", chainName, vaultID).Take(&chain).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "chain not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to load chain",
			})
			return
		}

		var request eventsRequest
		_ = c.ShouldBindQuery(&request)

		parentID, err := parseFetchCursor(request)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err":    "UNKNOWN",
				"errmsg": err.Error(),
			})
			return
		}

		limit := 100
		if requestLimit := c.Query("limit"); requestLimit != "" {
			parsedLimit, parseErr := strconv.Atoi(requestLimit)
			if parseErr != nil || parsedLimit <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "invalid limit",
				})
				return
			}
			if parsedLimit < limit {
				limit = parsedLimit
			}
		}

		events, err := fetchEventsAfterCursor(deps.DB, vaultID, chainName, parentID, limit)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"err":    "PARENT_EVENT_NOT_FOUND",
					"errmsg": "parent event not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to load events",
			})
			return
		}

		c.JSON(http.StatusOK, eventsResponse{Err: "0", Events: events})
	}
}

func LastEventV1(deps *api.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "database is not configured",
			})
			return
		}

		chainName := c.Param("chainName")

		// TODO: check for name

		tokenType, ok := auth.GetCurrentTokenType(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "unauthorized",
			})
			return
		}

		var accountID uuid.UUID
		var vaultID uuid.UUID

		if tokenType == auth.TokenTypeOauth {
			var scope string
			accountID, vaultID, scope, ok = auth.GetCurrentOauthContext(c, deps.DB)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "unauthorized",
				})
				return
			}

			parts := strings.Split(scope, ";")
			found := slices.Contains(parts, "chain:"+chainName+":r") || slices.Contains(parts, "chain:"+chainName+":rw")

			if !found {
				c.JSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "unauthorized",
				})
			}
		} else {
			accountID, ok = auth.GetCurrentAccount(c)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "unauthorized",
				})
				return
			}

			var err error
			vaultID, err = uuid.Parse(c.Param("vaultId"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "invalid vault id",
				})
				return
			}
		}

		if _, err := loadVaultForAccount(deps.DB, vaultID, accountID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "vault not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to load vault",
			})
			return
		}

		var chain models.Chain
		if err := deps.DB.Where("name = ? AND vault_id = ?", chainName, vaultID).Take(&chain).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "chain not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to load chain",
			})
			return
		}

		if chain.LastEventID == nil {
			c.JSON(http.StatusOK, gin.H{
				"err":    "NO_LAST_EVENT",
				"errmsg": "chain doesn't have last event",
			})
			return
		}

		var event models.Event
		if err := deps.DB.Where("vault_id = ? AND chain_name = ? AND id = ?", vaultID, chainName, *chain.LastEventID).Take(&event).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "last event not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to load last event",
			})
			return
		}

		c.JSON(http.StatusOK, lastEventResponse{Err: "0", Event: &event})
	}
}

func AddEventV1(deps *api.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "database is not configured",
			})
			return
		}

		chainName := c.Param("chainName")

		// TODO: check for name

		tokenType, ok := auth.GetCurrentTokenType(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "unauthorized",
			})
			return
		}

		var accountID uuid.UUID
		var vaultID uuid.UUID

		if tokenType == auth.TokenTypeOauth {
			var scope string
			accountID, vaultID, scope, ok = auth.GetCurrentOauthContext(c, deps.DB)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "unauthorized",
				})
				return
			}

			parts := strings.Split(scope, ";")
			found := slices.Contains(parts, "chain:"+chainName+":rw")

			if !found {
				c.JSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "unauthorized",
				})
			}
		} else {
			accountID, ok = auth.GetCurrentAccount(c)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "unauthorized",
				})
				return
			}

			var err error
			vaultID, err = uuid.Parse(c.Param("vaultId"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "invalid vault id",
				})
				return
			}
		}

		if _, err := loadVaultForAccount(deps.DB, vaultID, accountID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "vault not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to load vault",
			})
			return
		}

		var request addEventRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "invalid request body",
			})
			return
		}

		// TODO: add check that payload is Base64 and has "n" (name) and "e" (event) parameters

		eventID, err := uuid.Parse(request.EventID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "invalid event id",
			})
			return
		}

		parentID, err := misc.ParseOptionalUUID(request.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "invalid parent_id",
			})
			return
		}

		tx := deps.DB.Begin()
		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to start transaction",
			})
			return
		}

		rollback := func() {
			_ = tx.Rollback().Error
		}

		if _, err := loadVaultForAccount(tx, vaultID, accountID); err != nil {
			rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "vault not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to load vault",
			})
			return
		}

		var chain models.Chain

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ? AND vault_id = ?", chainName, vaultID).Take(&chain).Error; err != nil {
			rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "chain not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to load chain",
			})
			return
		}

		if !misc.SameOptionalUUID(parentID, chain.LastEventID) {
			rollback()
			c.JSON(http.StatusConflict, gin.H{
				"err":           "UNKNOWN",
				"errmsg":        "last event id is not same",
				"last_event_id": misc.OptionalUUIDToString(chain.LastEventID),
			})
			return
		}

		event := models.Event{
			ID:        eventID,
			ChainName: chainName,
			VaultID:   vaultID,
			ParentID:  parentID,
			Payload:   request.Payload,
			CreatedAt: time.Now(),
		}

		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{
			{Name: "id"},
			{Name: "chain_name"},
			{Name: "vault_id"},
		}, DoNothing: true}).Create(&event)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				rollback()
				c.JSON(http.StatusConflict, gin.H{
					"err":           "UNKNOWN",
					"errmsg":        "last event id is not same",
					"last_event_id": misc.OptionalUUIDToString(chain.LastEventID),
				})
				return
			}

			rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to insert event",
			})
			return
		}

		if result.RowsAffected == 0 {
			var existing models.Event
			if err := tx.Where("vault_id = ? AND chain_name = ? AND id = ?", vaultID, chainName, eventID).Take(&existing).Error; err != nil {
				rollback()
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusConflict, gin.H{
						"err":           "UNKNOWN",
						"errmsg":        "last event id is not same",
						"last_event_id": misc.OptionalUUIDToString(chain.LastEventID),
					})
					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "failed to check existing event",
				})
				return
			}

			if !misc.SameOptionalUUID(existing.ParentID, parentID) || existing.Payload != request.Payload {
				rollback()
				c.JSON(http.StatusConflict, gin.H{
					"err":           "UNKNOWN",
					"errmsg":        "last event id is not same",
					"last_event_id": misc.OptionalUUIDToString(chain.LastEventID),
				})
				return
			}

			if err := tx.Commit().Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "failed to complete transaction",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"err": "0",
			})
			return
		}

		// Handle special events
		if chainName == "tokens" {
			var payload struct {
				Name  string `json:"n"`
				Event struct {
					Token string `json:"id"`
					Scope string `json:"scope"`
				} `json:"e"`
			}

			if err := json.Unmarshal([]byte(request.Payload), &payload); err != nil {
				rollback()
				c.JSON(http.StatusBadRequest, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "invalid event payload",
				})
				return
			}

			if payload.Name == "issue_token_v1" {
				tokenID, err := uuid.Parse(payload.Event.Token)
				if err != nil {
					rollback()
					c.JSON(http.StatusBadRequest, gin.H{
						"err":    "UNKNOWN",
						"errmsg": "invalid oauth token id",
					})
					return
				}

				oauthToken := models.OauthToken{
					ID:        tokenID,
					VaultID:   vaultID,
					Scope:     payload.Event.Scope,
					CreatedAt: time.Now(),
				}

				if err := tx.Create(&oauthToken).Error; err != nil {
					rollback()

					if errors.Is(err, gorm.ErrDuplicatedKey) {
						c.JSON(http.StatusConflict, gin.H{
							"err":    "UNKNOWN",
							"errmsg": "oauth token already exists",
						})
						return
					}

					c.JSON(http.StatusInternalServerError, gin.H{
						"err":    "UNKNOWN",
						"errmsg": "failed to create oauth token",
					})
					return
				}
			}
		}

		if err := tx.Model(&models.Chain{}).Where("name = ? AND vault_id = ?", chainName, vaultID).Update("last_event_id", eventID).Error; err != nil {
			rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to update chain state",
			})
			return
		}

		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to complete transaction",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"err":    "0",
			"errmsg": "event created successfully",
		})
	}
}

func parseFetchCursor(request eventsRequest) (*uuid.UUID, error) {
	rawCursor := request.AfterID

	if rawCursor == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(rawCursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}

	return &parsed, nil
}

func fetchEventsAfterCursor(db *gorm.DB, vaultID uuid.UUID, chainName string, parentID *uuid.UUID, limit int) ([]models.Event, error) {
	if limit <= 0 {
		return []models.Event{}, nil
	}

	if parentID != nil {
		var base models.Event
		if err := db.Where("vault_id = ? AND chain_name = ? AND id = ?", vaultID, chainName, *parentID).Take(&base).Error; err != nil {
			return nil, err
		}
	}

	tableName, err := eventTableName(db)
	if err != nil {
		return nil, err
	}

	events := make([]models.Event, 0, limit)

	if parentID == nil {
		query := fmt.Sprintf(`
WITH RECURSIVE event_chain AS (
	(
		SELECT e.*
		FROM %s AS e
		WHERE e.vault_id = ? AND e.chain_name = ? AND e.parent_id IS NULL
		LIMIT 1
	)

    UNION ALL

    SELECT child.*
    FROM event_chain AS ec
    JOIN LATERAL (
        SELECT e.*
        FROM %s AS e
        WHERE e.vault_id = ? AND e.chain_name = ? AND e.parent_id = ec.id
        LIMIT 1
    ) AS child ON true
)
SELECT *
FROM event_chain
LIMIT ?`, tableName, tableName)

		if err := db.Raw(query, vaultID, chainName, vaultID, chainName, limit).Scan(&events).Error; err != nil {
			return nil, err
		}

		return events, nil
	}

	query := fmt.Sprintf(`
WITH RECURSIVE event_chain AS (
	(
		SELECT e.*
		FROM %s AS e
		WHERE e.vault_id = ? AND e.chain_name = ? AND e.id = ?
		LIMIT 1
	)

    UNION ALL

    SELECT child.*
    FROM event_chain AS ec
    JOIN LATERAL (
        SELECT e.*
        FROM %s AS e
        WHERE e.vault_id = ? AND e.chain_name = ? AND e.parent_id = ec.id
        LIMIT 1
    ) AS child ON true
)
SELECT *
FROM event_chain
WHERE id <> ?
LIMIT ?`, tableName, tableName)

	if err := db.Raw(query, vaultID, chainName, *parentID, vaultID, chainName, *parentID, limit).Scan(&events).Error; err != nil {
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
