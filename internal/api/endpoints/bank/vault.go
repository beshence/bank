package bank

import (
	"bank/internal/api"
	"bank/internal/auth"
	"bank/internal/database/models"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type vaultRequest struct {
	Name string `json:"name" binding:"required,min=1,max=128"`
}

type vaultResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func VaultsV1(deps *api.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": "UNKNOWN", "errmsg": "database is not configured"})
			return
		}

		accountID, ok := auth.GetCurrentAccount(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "unauthorized",
			})
			return
		}

		vaults := make([]models.Vault, 0)
		if err := deps.DB.Where("account_id = ?", accountID).Order("created_at desc").Find(&vaults).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": "UNKNOWN", "errmsg": "failed to load vaults"})
			return
		}

		items := make([]vaultResponse, len(vaults))
		for i, item := range vaults {
			items[i] = vaultResponse{ID: item.ID.String(), Name: item.Name}
		}

		c.JSON(http.StatusOK, gin.H{"err": "0", "vaults": items})
	}
}

func CreateVaultV1(deps *api.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "database is not configured",
			})
			return
		}

		accountID, ok := auth.GetCurrentAccount(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "unauthorized",
			})
			return
		}

		var request vaultRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "invalid request body",
			})
			return
		}

		vault := models.Vault{Name: request.Name, AccountID: accountID}
		if err := deps.DB.Create(&vault).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				c.JSON(http.StatusConflict, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "you already have a vault with this name",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to create vault",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"err":  "0",
			"id":   vault.ID.String(),
			"name": vault.Name,
		})
	}
}

func loadVaultForAccount(db *gorm.DB, vaultID uuid.UUID, accountID uuid.UUID) (models.Vault, error) {
	var vault models.Vault
	if err := db.Where("id = ? AND account_id = ?", vaultID, accountID).Take(&vault).Error; err != nil {
		return models.Vault{}, err
	}

	return vault, nil
}
