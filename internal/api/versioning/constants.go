package versioning

import (
	"bank/internal/api"
	authend "bank/internal/api/endpoints/auth"
	"bank/internal/api/endpoints/bank"
	"bank/internal/api/endpoints/misc"
	"bank/internal/auth"
	"net/http"
)

const (
	HeaderAPIVersion  = "X-Beshence-Bank-API-Version"
	VersionV1dot0dot0 = "v1.0.0"
	DefaultAPIVersion = VersionV1dot0dot0
)

var supportedVersions = []string{VersionV1dot0dot0 /*, VersionV1dot1*/}

var versionIndex = map[string]int{
	VersionV1dot0dot0: 0,
	// VersionV1dot1dot0: 1,
}

func GetVersionedEndpoints(deps *api.Dependencies) VersionedEndpoints {
	return VersionedEndpoints{
		VersionV1dot0dot0: {
			http.MethodGet: {
				"/ping":                  misc.PingV1(deps, supportedVersions),
				"/ek":                    misc.EKV1(deps),
				"/auth/me":               auth.RequireAuth(deps.AccessJWTManager, authend.MeV1(deps)),
				"/auth/refresh":          auth.RequireAuth(deps.RefreshJWTManager, authend.RefreshV1(deps)),
				"/vaults":                auth.RequireAuth(deps.AccessJWTManager, bank.VaultsV1(deps)),
				"/vault/:vaultId/chains": auth.RequireAuth(deps.AccessJWTManager, bank.ChainsV1(deps)),
				"/vault/:vaultId/" +
					"chain/:chainName/events": auth.RequireAuth(deps.AccessJWTManager, bank.EventsV1(deps)),
				"/vault/:vaultId/" +
					"chain/:chainName/event/last": auth.RequireAuth(deps.AccessJWTManager, bank.LastEventV1(deps)),
			},
			http.MethodPost: {
				"/auth/register":        authend.RegisterV1(deps),
				"/auth/login":           authend.LoginV1(deps),
				"/vault":                auth.RequireAuth(deps.AccessJWTManager, bank.CreateVaultV1(deps)),
				"/vault/:vaultId/chain": auth.RequireAuth(deps.AccessJWTManager, bank.CreateChainV1(deps)),
				"/vault/:vaultId/" +
					"chain/:chainName/event": auth.RequireAuth(deps.AccessJWTManager, bank.AddEventV1(deps)),
			},
		},
	}
}
