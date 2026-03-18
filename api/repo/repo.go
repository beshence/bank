package repo

import (
	"errors"
	"net/http"

	"vault/internal/app"
	"vault/internal/database/models"
	"vault/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createRepositoryRequest struct {
	Name string `json:"name" binding:"required,min=1,max=128"`
}

func CreateRepoV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "database is not configured",
			})
			return
		}

		userID, _, ok := middleware.GetCurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		var request createRepositoryRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "invalid request body",
			})
			return
		}

		repository := models.Repository{
			Name:  request.Name,
			Owner: userID,
		}

		if err := repository.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

		if err := deps.DB.Create(&repository).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				c.JSON(http.StatusConflict, gin.H{
					"message": "you already have a repository with this name",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to create repository",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"repository": repository,
		})
	}
}

func ReposV1dot0(deps *app.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "database is not configured",
			})
			return
		}

		userID, _, ok := middleware.GetCurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		repositories := make([]models.Repository, 0)
		if err := deps.DB.Where("owner = ?", userID).Order("created_at desc").Find(&repositories).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to load repositories",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"repositories": repositories,
		})
	}
}
