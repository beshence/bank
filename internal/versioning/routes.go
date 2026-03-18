package versioning

import "github.com/gin-gonic/gin"

func RegisterVersionedRoute(g *gin.RouterGroup, handlersByVersion map[string]EndpointHandlers, method string, endpoint string, middlewares ...gin.HandlerFunc) {
	handlers := append(middlewares, RouteByVersion(handlersByVersion, endpoint))
	g.Handle(method, endpoint, handlers...)
}
