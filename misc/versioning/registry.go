package versioning

import (
	"vault/api/auth"
	"vault/api/misc"

	"github.com/gin-gonic/gin"
)

type EndpointHandlers map[string]gin.HandlerFunc

var HandlersByVersion = map[string]EndpointHandlers{
	VersionV1: {
		EndpointPing:     misc.PingV1,
		EndpointRegister: auth.RegisterV1,
		EndpointLogin:    auth.LoginV1,
	},
}
