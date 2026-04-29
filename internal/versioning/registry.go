package versioning

import (
	"net/http"

	"vault/api/auth"
	"vault/api/misc"
	"vault/api/repo"
	"vault/internal/app"

	"github.com/gin-gonic/gin"
)

type MethodHandlers map[string]gin.HandlerFunc
type EndpointHandlers map[string]MethodHandlers

func NewHandlersByVersion(deps *app.Dependencies) map[string]EndpointHandlers {
	return map[string]EndpointHandlers{
		VersionV1dot0: {
			http.MethodGet: {
				EndpointPing:       misc.PingV1dot0,
				EndpointMe:         auth.MeV1dot0(deps),
				EndpointRepo:       repo.ReposV1dot0(deps),
				EndpointRepoEvents: repo.FetchEventsV1dot0(deps),
			},
			http.MethodPost: {
				EndpointRegister:   auth.RegisterV1dot0(deps),
				EndpointLogin:      auth.LoginV1dot0(deps),
				EndpointRefresh:    auth.RefreshV1dot0(deps),
				EndpointRepo:       repo.CreateRepoV1dot0(deps),
				EndpointRepoEvents: repo.AppendEventV1dot0(deps),
			},
		},
	}
}
