package versioning

import (
	"vault/api/auth"
	"vault/api/misc"
	"vault/internal/app"

	"github.com/gin-gonic/gin"
)

type EndpointHandlers map[string]gin.HandlerFunc

func NewHandlersByVersion(deps *app.Dependencies) map[string]EndpointHandlers {
	return map[string]EndpointHandlers{
		VersionV1dot0: {
			EndpointPing:     misc.PingV1dot0,
			EndpointRegister: auth.RegisterV1dot0(deps),
			EndpointLogin:    auth.LoginV1dot0(deps),
			EndpointMe:       auth.MeV1dot0(deps),
		},
	}
}
