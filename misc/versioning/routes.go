package versioning

import "github.com/gin-gonic/gin"

func RegisterVersionedRoute(g *gin.RouterGroup, method string, endpoint string) {
	g.Handle(method, endpoint, RouteByVersion(endpoint))
}
