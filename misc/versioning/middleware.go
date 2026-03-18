package versioning

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RouteByVersion(endpoint string) gin.HandlerFunc {
	return func(c *gin.Context) {
		version := c.GetHeader(HeaderAPIVersion)
		if version == "" {
			version = DefaultAPIVersion
		}

		handlers, ok := handlersByVersion[version]
		if !ok {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"message": fmt.Sprintf("unsupported API version: %s", version),
			})
			return
		}

		handler, ok := handlers[endpoint]
		if !ok {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"message": fmt.Sprintf("endpoint %s is not available for version %s", endpoint, version),
			})
			return
		}

		handler(c)
	}
}
