package versioning

const (
	HeaderAPIVersion  = "X-Vault-API-Version"
	VersionV1         = "v1"
	DefaultAPIVersion = VersionV1
)

const (
	EndpointPing     = "/ping"
	EndpointRegister = "/auth/register"
	EndpointLogin    = "/auth/login"
)
