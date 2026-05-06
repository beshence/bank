package middleware

import "github.com/gin-gonic/gin"

const (
	ContextAuthClaimsKey         = "auth.claims"
	ContextAuthSessionIDKey      = "auth.session_id"
	ContextAuthAccountIDKey      = "auth.account_id"
	ContextAuthRefreshTokenIDKey = "auth.refresh_token_id"
)

func GetCurrentAccount(c *gin.Context) (string, bool) {
	accountIDValue, accountIDExists := c.Get(ContextAuthAccountIDKey)
	if !accountIDExists {
		return "", false
	}

	accountID, accountIDOk := accountIDValue.(string)
	if !accountIDOk || accountID == "" {
		return "", false
	}

	return accountID, true
}

func GetCurrentSession(c *gin.Context) (string, string, bool) {
	sessionIDValue, sessionIDExists := c.Get(ContextAuthSessionIDKey)
	refreshTokenIDValue, refreshTokenIDExists := c.Get(ContextAuthRefreshTokenIDKey)
	if !sessionIDExists || !refreshTokenIDExists {
		return "", "", false
	}

	sessionID, sessionIDOk := sessionIDValue.(string)
	refreshTokenID, refreshTokenIDOk := refreshTokenIDValue.(string)
	if !sessionIDOk || !refreshTokenIDOk || sessionID == "" || refreshTokenID == "" {
		return "", "", false
	}

	return sessionID, refreshTokenID, true
}
