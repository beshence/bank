package security

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret []byte
	ttl    time.Duration
	typeID TokenType
}

type AuthClaims struct {
	SessionID     string
	AccountID     string
	AccessTokenID string
	TokenType     TokenType
}

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

func NewJWTManager(secret string, ttl time.Duration, typeID TokenType) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
		ttl:    ttl,
		typeID: typeID,
	}
}

func (m *JWTManager) GenerateToken(sessionID string, accountID string, accessTokenID string) (string, int64, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.ttl)

	claims := jwt.MapClaims{
		"sub":  sessionID,
		"aid":  accountID,
		"atid": accessTokenID,
		"typ":  string(m.typeID),
		"iat":  now.Unix(),
		"exp":  expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", 0, err
	}

	return signedToken, expiresAt.Unix(), nil
}

func (m *JWTManager) ParseToken(tokenString string) (jwt.MapClaims, error) {
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

func AuthClaimsFromToken(claims jwt.MapClaims) (AuthClaims, bool) {
	tokenType, ok := TokenTypeFromClaims(claims)
	if !ok {
		return AuthClaims{}, false
	}

	sessionID, sessionIDOk := claims["sub"].(string)
	accountID, accountIDOk := claims["aid"].(string)
	accessTokenID, accessTokenIDOk := claims["atid"].(string)
	if !sessionIDOk || !accountIDOk || !accessTokenIDOk {
		return AuthClaims{}, false
	}

	if sessionID == "" || accountID == "" || accessTokenID == "" {
		return AuthClaims{}, false
	}

	return AuthClaims{
		SessionID:     sessionID,
		AccountID:     accountID,
		AccessTokenID: accessTokenID,
		TokenType:     tokenType,
	}, true
}
