package vault

import (
	"errors"
	"net/http"

	"vault/internal/app"
	"vault/internal/database/models"
	"vault/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type vaultRequest struct {
	Name string `json:"name" binding:"required,min=1,max=128"`
}

type vaultResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func VaultsV1dot0(deps *app.Dependencies) gin.HandlerFunc {
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

		vaults := make([]models.Vault, 0)
		if err := deps.DB.Where("account_id = ?", accountUUID).Order("created_at desc").Find(&vaults).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load vaults"})
			return
		}

		items := make([]vaultResponse, len(vaults))
		for i, item := range vaults {
			items[i] = vaultResponse{ID: item.ID.String(), Name: item.Name, CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}
		}

		c.JSON(http.StatusOK, gin.H{"vaults": items})
	}
}

func CreateVaultV1dot0(deps *app.Dependencies) gin.HandlerFunc {
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

		var request vaultRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		vault := models.Vault{Name: request.Name, AccountID: accountUUID}
		if err := deps.DB.Create(&vault).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				c.JSON(http.StatusConflict, gin.H{"message": "you already have a vault with this name"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create vault"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"vault": vaultResponse{ID: vault.ID.String(), Name: vault.Name, CreatedAt: vault.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}})
	}
}
