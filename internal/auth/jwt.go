package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWT struct {
	secret    []byte
	ttl       time.Duration
	tokenType TokenType
}

type Claims struct {
	SessionID      string
	AccountID      string
	RefreshTokenID string
	TokenType      TokenType
}

type TokenType string

const (
	TokenTypeOauth   TokenType = "oauth"
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

func NewJWTManager(secret string, ttl time.Duration, tokenType TokenType) *JWT {
	return &JWT{
		secret:    []byte(secret),
		ttl:       ttl,
		tokenType: tokenType,
	}
}

func (m *JWT) GenerateToken(sessionID uuid.UUID, accountID uuid.UUID, refreshTokenID uuid.UUID) (string, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.ttl)

	claims := jwt.MapClaims{
		"sub": sessionID.String(),
		"aid": accountID.String(),
		"typ": string(m.tokenType),
		"iat": now.Unix(),
		"exp": expiresAt.Unix(),
	}

	if m.tokenType == TokenTypeRefresh {
		claims["rid"] = refreshTokenID.String()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func (m *JWT) ParseToken(tokenString string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}

	parsedToken, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}

	if !parsedToken.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}

func TokenTypeFromClaims(claims jwt.MapClaims) (TokenType, bool) {
	typeValue, ok := claims["typ"].(string)
	if !ok || typeValue == "" {
		return "", false
	}

	tokenType := TokenType(typeValue)
	if tokenType != TokenTypeAccess && tokenType != TokenTypeRefresh {
		return "", false
	}

	return tokenType, true
}

func ClaimsFromToken(claims jwt.MapClaims) (Claims, bool) {
	tokenType, ok := TokenTypeFromClaims(claims)
	if !ok {
		return Claims{}, false
	}

	sessionID, sessionIDOk := claims["sub"].(string)
	accountID, accountIDOk := claims["aid"].(string)
	if !sessionIDOk || !accountIDOk {
		return Claims{}, false
	}

	if sessionID == "" || accountID == "" {
		return Claims{}, false
	}

	refreshTokenID := ""
	if tokenType == TokenTypeRefresh {
		var refreshTokenIDOk bool
		refreshTokenID, refreshTokenIDOk = claims["rid"].(string)
		if !refreshTokenIDOk || refreshTokenID == "" {
			return Claims{}, false
		}
	}

	return Claims{
		SessionID:      sessionID,
		AccountID:      accountID,
		RefreshTokenID: refreshTokenID,
		TokenType:      tokenType,
	}, true
}
