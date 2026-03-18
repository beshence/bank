package versioning

const (
	HeaderAPIVersion  = "X-Vault-API-Version"
	VersionV1dot0     = "v1.0"
	DefaultAPIVersion = VersionV1dot0
)

const (
	EndpointPing     = "/ping"
	EndpointRegister = "/auth/register"
	EndpointLogin    = "/auth/login"
	EndpointRefresh  = "/auth/refresh"
	EndpointMe       = "/auth/me"
	EndpointRepo     = "/repo"
)
