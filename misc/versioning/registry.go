package versioning

import (
	"vault/api/auth"
	"vault/api/misc"

	"github.com/gin-gonic/gin"
)

type endpointHandlers map[string]gin.HandlerFunc

var handlersByVersion = map[string]endpointHandlers{
	VersionV1: {
		EndpointPing:     misc.PingV1,
		EndpointRegister: auth.RegisterV1,
		EndpointLogin:    auth.LoginV1,
	},
}
