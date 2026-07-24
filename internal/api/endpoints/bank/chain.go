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

type createChainRequest struct {
	Name string `json:"name" binding:"required,min=1,max=128"`
}

type chainResponse struct {
	Name string `json:"name"`
}

func ChainsV1(deps *api.Dependencies) gin.HandlerFunc {
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

		vaultID, err := uuid.Parse(c.Param("vaultId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "invalid vault id",
			})
			return
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

		chains := make([]models.Chain, 0)
		if err := deps.DB.Where("vault_id = ?", vaultID).Order("created_at desc").Find(&chains).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": "UNKNOWN", "errmsg": "failed to load chains"})
			return
		}

		items := make([]chainResponse, len(chains))
		for i, chain := range chains {
			items[i] = chainResponse{Name: chain.Name}
		}

		c.JSON(http.StatusOK, gin.H{
			"err":    "0",
			"chains": items,
		})
	}
}

func CreateChainV1(deps *api.Dependencies) gin.HandlerFunc {
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

		vaultID, err := uuid.Parse(c.Param("vaultId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "invalid vault id",
			})
			return
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

		var request createChainRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "invalid request body",
			})
			return
		}

		chain := models.Chain{
			Name:    request.Name,
			VaultID: vaultID,
		}

		if err := chain.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err":     "UNKNOWN",
				"message": err.Error(),
			})
			return
		}

		if err := deps.DB.Create(&chain).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				c.JSON(http.StatusConflict, gin.H{
					"err":    "UNKNOWN",
					"errmsg": "you already have a chain with this name in this bank",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "failed to create chain",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"err":  "0",
			"name": chain.Name,
		})
	}
}
