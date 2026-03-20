package middleware

import "github.com/gin-gonic/gin"

const (
	ContextAuthClaimsKey        = "auth.claims"
	ContextAuthSessionIDKey     = "auth.session_id"
	ContextAuthUserIDKey        = "auth.user_id"
	ContextAuthAccessTokenIDKey = "auth.access_token_id"
)

func GetCurrentUser(c *gin.Context) (string, bool) {
	userIDValue, userIDExists := c.Get(ContextAuthUserIDKey)
	if !userIDExists {
		return "", false
	}

	userID, userIDOk := userIDValue.(string)
	if !userIDOk || userID == "" {
		return "", false
	}

	return userID, true
}

func GetCurrentSession(c *gin.Context) (string, string, bool) {
	sessionIDValue, sessionIDExists := c.Get(ContextAuthSessionIDKey)
	accessTokenIDValue, accessTokenIDExists := c.Get(ContextAuthAccessTokenIDKey)
	if !sessionIDExists || !accessTokenIDExists {
		return "", "", false
	}

	sessionID, sessionIDOk := sessionIDValue.(string)
	accessTokenID, accessTokenIDOk := accessTokenIDValue.(string)
	if !sessionIDOk || !accessTokenIDOk || sessionID == "" || accessTokenID == "" {
		return "", "", false
	}

	return sessionID, accessTokenID, true
}
