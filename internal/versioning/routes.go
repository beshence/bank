package versioning

import "github.com/gin-gonic/gin"

func RegisterVersionedRoute(g *gin.RouterGroup, handlersByVersion map[string]EndpointHandlers, method string, endpoint string, middlewares ...gin.HandlerFunc) {
	handlers := make([]gin.HandlerFunc, len(middlewares), len(middlewares)+1)
	copy(handlers, middlewares)
	handlers = append(handlers, RouteByVersion(handlersByVersion, endpoint))
	g.Handle(method, endpoint, handlers...)
}
