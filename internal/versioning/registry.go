package versioning

import (
	"net/http"

	"bank/api/auth"
	"bank/api/bank"
	"bank/api/chain"
	"bank/api/misc"
	"bank/internal/app"

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
				EndpointVaults:           bank.VaultsV1dot0(deps),
				EndpointVaultChains:      chain.ChainsV1dot0(deps),
				EndpointVaultChainEvents: chain.FetchEventsV1dot0(deps),
			},
			http.MethodPost: {
				EndpointRegister:         auth.RegisterV1dot0(deps),
				EndpointLogin:            auth.LoginV1dot0(deps),
				EndpointRefresh:          auth.RefreshV1dot0(deps),
				EndpointVaults:           bank.CreateVaultV1dot0(deps),
				EndpointVaultChains:      chain.CreateChainV1dot0(deps),
				EndpointVaultChainEvents: chain.AppendEventV1dot0(deps),
			},
		},
	}
}
