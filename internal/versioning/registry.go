package versioning

import (
	"net/http"

	"vault/api/auth"
	"vault/api/chain"
	"vault/api/misc"
	"vault/api/vault"
	"vault/internal/app"

	"github.com/gin-gonic/gin"
)

type MethodHandlers map[string]gin.HandlerFunc
type EndpointHandlers map[string]MethodHandlers

func NewHandlersByVersion(deps *app.Dependencies) map[string]EndpointHandlers {
	return map[string]EndpointHandlers{
		VersionV1dot0: {
			http.MethodGet: {
				EndpointPing:             misc.PingV1dot0,
				EndpointMe:               auth.MeV1dot0(deps),
				EndpointVaults:           vault.VaultsV1dot0(deps),
				EndpointVaultChains:      chain.ChainsV1dot0(deps),
				EndpointVaultChainEvents: chain.FetchEventsV1dot0(deps),
			},
			http.MethodPost: {
				EndpointRegister:         auth.RegisterV1dot0(deps),
				EndpointLogin:            auth.LoginV1dot0(deps),
				EndpointRefresh:          auth.RefreshV1dot0(deps),
				EndpointVaults:           vault.CreateVaultV1dot0(deps),
				EndpointVaultChains:      chain.CreateChainV1dot0(deps),
				EndpointVaultChainEvents: chain.AppendEventV1dot0(deps),
			},
		},
	}
}
