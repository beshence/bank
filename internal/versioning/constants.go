package versioning

const (
	HeaderAPIVersion  = "X-Beshence-Bank-API-Version"
	VersionV1dot0     = "v1.0"
	DefaultAPIVersion = VersionV1dot0
)

const (
	EndpointPing             = "/ping"
	EndpointRegister         = "/auth/register"
	EndpointLogin            = "/auth/login"
	EndpointRefresh          = "/auth/refresh"
	EndpointMe               = "/auth/me"
	EndpointVaults           = "/vaults"
	EndpointVaultChains      = "/vaults/:vaultId/chains"
	EndpointVaultChainEvents = "/vaults/:vaultId/chains/:chainId/events"
)
