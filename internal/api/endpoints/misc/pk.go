package misc

import (
	"bank/internal/api"
	"bank/internal/settings"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetPublicKeysV1(deps *api.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"err": "UNKNOWN", "errmsg": "database is not configured"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"err":  "0",
			"root": settings.GetBankRootPublicKeyBase64(deps.DB),
			"leaf": gin.H{
				"pk":  "",
				"sig": "",
			},
		})
	}
}
