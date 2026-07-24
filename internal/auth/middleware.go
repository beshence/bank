package auth

import (
	"bank/internal/database/models"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

func GetCurrentOauthContext(c *gin.Context, db *gorm.DB) (uuid.UUID, uuid.UUID, string, bool) {
	tokenValue, tokenExists := c.Get(ContextAuthOauthTokenKey)
	if !tokenExists {
		return uuid.Nil, uuid.Nil, "", false
	}

	token, tokenOk := tokenValue.(string)
	if !tokenOk || token == "" {
		return uuid.Nil, uuid.Nil, "", false
	}

	// the token is in the format "oauthv1_<accountID>_<vaultID>_<tokenID>"
	parts := strings.Split(token, "_")
	if len(parts) != 4 || parts[0] != "oauthv1" {
		return uuid.Nil, uuid.Nil, "", false
	}

	accountID, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, uuid.Nil, "", false
	}

	vaultID, err := uuid.Parse(parts[2])
	if err != nil {
		return uuid.Nil, uuid.Nil, "", false
	}

	tokenID, err := uuid.Parse(parts[3])
	if err != nil {
		return uuid.Nil, uuid.Nil, "", false
	}

	var oauthToken models.OauthToken

	err = db.Where("id = ? AND account_id = ? AND vault_id = ?",
		tokenID, accountID, vaultID).First(&oauthToken).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, uuid.Nil, "", false
	} else if err != nil {
		return uuid.Nil, uuid.Nil, "", false
	}

	return accountID, vaultID, oauthToken.Scope, true
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

func RequireAuth(jwt *JWT, db *gorm.DB, h gin.HandlerFunc) gin.HandlerFunc {
	mw := CheckAuth(jwt, db)
	return func(c *gin.Context) {
		mw(c)
		if c.IsAborted() {
			return
		}
		h(c)
	}
}

func CheckAuth(jwtManager *JWT, db *gorm.DB) gin.HandlerFunc {
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

			// the token is in the format "oauthv1_<accountID>_<vaultID>_<tokenID>"
			parts := strings.Split(token, "_")
			if len(parts) != 4 || parts[0] != "oauthv1" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "invalid oauth token",
				})
				return
			}

			accountID, err := uuid.Parse(parts[1])
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "invalid oauth token",
				})
				return
			}

			vaultID, err := uuid.Parse(parts[2])
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "invalid oauth token",
				})
				return
			}

			tokenID, err := uuid.Parse(parts[3])
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "invalid oauth token",
				})
				return
			}

			var oauthToken models.OauthToken

			err = db.Where("id = ? AND account_id = ? AND vault_id = ?",
				tokenID, accountID, vaultID).First(&oauthToken).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"err":    "UNAUTHORIZED",
					"errmsg": "invalid oauth token",
				})
				return
			} else if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"err":    "INTERNAL_SERVER_ERROR",
					"errmsg": "database error",
				})
				return
			}

			c.Set(ContextAuthTokenTypeKey, TokenTypeOauth)
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

			c.Set(ContextAuthTokenTypeKey, jwtManager.tokenType)
			c.Set(ContextAuthClaimsKey, claims)
			c.Set(ContextAuthSessionIDKey, sessionID)
			c.Set(ContextAuthAccountIDKey, accountID)
			c.Set(ContextAuthRefreshTokenIDKey, refreshTokenID)
			c.Next()
		}
	}
}
