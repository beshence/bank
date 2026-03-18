package middleware

import "github.com/gin-gonic/gin"

const (
	ContextAuthClaimsKey = "auth.claims"
	ContextAuthUserIDKey = "auth.user_id"
	ContextAuthLoginKey  = "auth.login"
)

func GetCurrentUser(c *gin.Context) (string, string, bool) {
	userIDValue, userIDExists := c.Get(ContextAuthUserIDKey)
	loginValue, loginExists := c.Get(ContextAuthLoginKey)
	if !userIDExists || !loginExists {
		return "", "", false
	}

	userID, userIDOk := userIDValue.(string)
	login, loginOk := loginValue.(string)
	if !userIDOk || !loginOk || userID == "" || login == "" {
		return "", "", false
	}

	return userID, login, true
}
