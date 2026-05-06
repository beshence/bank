package chain

import (
	"errors"
	"net/http"
	"time"

	"vault/internal/app"
	"vault/internal/database/models"
	"vault/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createChainRequest struct {
	Name string `json:"name" binding:"required,min=1,max=128"`
}

type chainResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func CreateChainV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "database is not configured",
			})
			return
		}

		accountID, ok := middleware.GetCurrentAccount(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		vaultOwnerID, err := uuid.Parse(accountID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		vaultID, err := uuid.Parse(c.Param("vaultId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "invalid vault id",
			})
			return
		}

		if _, err := loadVaultForAccount(deps.DB, vaultID, vaultOwnerID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"message": "vault not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to load vault",
			})
			return
		}

		var request createChainRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "invalid request body",
			})
			return
		}

		chain := models.Chain{
			Name:    request.Name,
			VaultID: vaultID,
		}

		if err := chain.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

		if err := deps.DB.Create(&chain).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				c.JSON(http.StatusConflict, gin.H{
					"message": "you already have a chain with this name in this vault",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to create chain",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"chain": chainResponse{ID: chain.ID.String(), Name: chain.Name, CreatedAt: chain.CreatedAt.Format(time.RFC3339)},
		})
	}
}

func ChainsV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "database is not configured",
			})
			return
		}

		accountID, ok := middleware.GetCurrentAccount(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		vaultOwnerID, err := uuid.Parse(accountID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		vaultID, err := uuid.Parse(c.Param("vaultId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "invalid vault id",
			})
			return
		}

		if _, err := loadVaultForAccount(deps.DB, vaultID, vaultOwnerID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"message": "vault not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to load vault",
			})
			return
		}

		chains := make([]models.Chain, 0)
		if err := deps.DB.Where("vault_id = ?", vaultID).Order("created_at desc").Find(&chains).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to load chains",
			})
			return
		}

		items := make([]chainResponse, len(chains))
		for i, chain := range chains {
			items[i] = chainResponse{ID: chain.ID.String(), Name: chain.Name, CreatedAt: chain.CreatedAt.Format(time.RFC3339)}
		}

		c.JSON(http.StatusOK, gin.H{
			"chains": items,
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
