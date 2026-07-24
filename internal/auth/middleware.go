package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextAuthClaimsKey         = "auth.claims"
	ContextAuthSessionIDKey      = "auth.session_id"
	ContextAuthAccountIDKey      = "auth.account_id"
	ContextAuthRefreshTokenIDKey = "auth.refresh_token_id"
)

func GetCurrentAccount(c *gin.Context) (uuid.UUID, bool) {
	accountIDValue, accountIDExists := c.Get(ContextAuthAccountIDKey)
	if !accountIDExists {
		return uuid.Nil, false
	}

	accountID, accountIDOk := accountIDValue.(uuid.UUID)
	if !accountIDOk || accountID == uuid.Nil {
		return uuid.Nil, false
	}

	return accountID, true
}

func GetCurrentSession(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	sessionIDValue, sessionIDExists := c.Get(ContextAuthSessionIDKey)
	refreshTokenIDValue, refreshTokenIDExists := c.Get(ContextAuthRefreshTokenIDKey)
	if !sessionIDExists || !refreshTokenIDExists {
		return uuid.Nil, uuid.Nil, false
	}

	sessionID, sessionIDOk := sessionIDValue.(uuid.UUID)
	refreshTokenID, refreshTokenIDOk := refreshTokenIDValue.(uuid.UUID)
	if !sessionIDOk || !refreshTokenIDOk || sessionID == uuid.Nil || refreshTokenID == uuid.Nil {
		return uuid.Nil, uuid.Nil, false
	}

	return sessionID, refreshTokenID, true
}

func RequireAuth(jwt *JWT, h gin.HandlerFunc) gin.HandlerFunc {
	mw := CheckAuth(jwt)
	return func(c *gin.Context) {
		mw(c)
		if c.IsAborted() {
			return
		}
		h(c)
	}
}

func CheckAuth(jwtManager *JWT) gin.HandlerFunc {
	return func(c *gin.Context) {
		if jwtManager == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"err":    "UNKNOWN",
				"errmsg": "auth is not configured",
			})
			return
		}

		authorizationHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authorizationHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "missing or invalid authorization header",
			})
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authorizationHeader, "Bearer "))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "missing bearer token",
			})
			return
		}

		claims, err := jwtManager.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "invalid token",
			})
			return
		}

		authClaims, claimsOk := ClaimsFromToken(claims)
		if !claimsOk {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "invalid token claims",
			})
			return
		}

		sessionID, err := uuid.Parse(authClaims.SessionID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "invalid token claims",
			})
			return
		}

		accountID, err := uuid.Parse(authClaims.AccountID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "invalid token claims",
			})
			return
		}

		refreshTokenID := uuid.Nil
		if authClaims.RefreshTokenID != "" {
			refreshTokenID, err = uuid.Parse(authClaims.RefreshTokenID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "invalid token claims",
				})
				return
			}
		}

		if authClaims.TokenType != jwtManager.tokenType {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"err":    "UNAUTHORIZED",
				"errmsg": "invalid token type",
			})
			return
		}

		c.Set(ContextAuthClaimsKey, claims)
		c.Set(ContextAuthSessionIDKey, sessionID)
		c.Set(ContextAuthAccountIDKey, accountID)
		c.Set(ContextAuthRefreshTokenIDKey, refreshTokenID)
		c.Next()
	}
}
