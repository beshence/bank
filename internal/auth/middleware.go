package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextAuthTokenTypeKey      = "auth.token_type"
	ContextAuthClaimsKey         = "auth.claims"
	ContextAuthSessionIDKey      = "auth.session_id"
	ContextAuthAccountIDKey      = "auth.account_id"
	ContextAuthRefreshTokenIDKey = "auth.refresh_token_id"
	ContextAuthOauthTokenKey     = "auth.oauth_token"
)

func GetCurrentTokenType(c *gin.Context) (TokenType, bool) {
	tokenTypeValue, tokenTypeExists := c.Get(ContextAuthTokenTypeKey)
	if !tokenTypeExists {
		return "", false
	}

	tokenType, tokenTypeOk := tokenTypeValue.(TokenType)
	if !tokenTypeOk || tokenType == "" {
		return "", false
	}

	return tokenType, true
}

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

		if strings.HasPrefix(token, "oauthv1_") {
			// this is an oauth token
			c.Set(ContextAuthOauthTokenKey, TokenTypeOauth)
			c.Set(ContextAuthOauthTokenKey, token)
			c.Next()
		} else {
			// this is a jwt token
			jwtClaims, err := jwtManager.ParseToken(token)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "invalid token",
				})
				return
			}

			claims, claimsOk := ClaimsFromToken(jwtClaims)
			if !claimsOk {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "invalid token claims",
				})
				return
			}

			sessionID, err := uuid.Parse(claims.SessionID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "invalid token claims",
				})
				return
			}

			accountID, err := uuid.Parse(claims.AccountID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "invalid token claims",
				})
				return
			}

			refreshTokenID := uuid.Nil
			if claims.RefreshTokenID != "" {
				refreshTokenID, err = uuid.Parse(claims.RefreshTokenID)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"err":    "UNAUTHORIZED",
						"errmsg": "invalid token claims",
					})
					return
				}
			}

			if claims.TokenType != jwtManager.tokenType {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "invalid token type",
				})
				return
			}

			c.Set(ContextAuthOauthTokenKey, jwtManager.tokenType)
			c.Set(ContextAuthClaimsKey, claims)
			c.Set(ContextAuthSessionIDKey, sessionID)
			c.Set(ContextAuthAccountIDKey, accountID)
			c.Set(ContextAuthRefreshTokenIDKey, refreshTokenID)
			c.Next()
		}
	}
}
