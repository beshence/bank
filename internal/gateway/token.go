package gateway

import (
	"bank/internal/settings"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/cloudflare/circl/sign/slhdsa"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

var (
	tokenMu sync.Mutex

	cachedToken string
	tokenExpiry time.Time
)

type challengeResponse struct {
	Nonce string `json:"nonce"`
}

type tokenResponse struct {
	AccessToken string `json:"token"`
}

func GetGatewayToken(db *gorm.DB) (string, error) {
	gatewayURL := "https://gateway.beshence.com/api"
	bankID := settings.GetBankID(db)

	tokenMu.Lock()
	defer tokenMu.Unlock()

	if cachedToken != "" && time.Until(tokenExpiry) > time.Minute {
		return cachedToken, nil
	}

	token, err := requestNewGatewayToken(db, gatewayURL, bankID)

	if err != nil {
		return "", err
	}

	cachedToken = token
	tokenExpiry, _ = parseTokenExpiry(token)

	return cachedToken, nil
}

func requestNewGatewayToken(
	db *gorm.DB,
	gatewayURL string,
	bankID string,
) (string, error) {
	privateKey, publicKey, err := settings.GetBankKeypair(db)
	publicKeyBytes, _ := publicKey.MarshalBinary()

	if err != nil {
		return "", err
	}

	// 1. publish EK

	publicKeyB64 := base64.RawURLEncoding.EncodeToString(publicKeyBytes)

	_, err = post(
		gatewayURL+"/bank/"+bankID+"/pk",
		map[string]string{
			"pk": publicKeyB64,
		},
		"",
	)

	if err != nil {
		return "", err
	}

	// 2. get challenge

	challengeRaw, err := get(
		gatewayURL+"/bank/"+bankID+"/challenge",
		"",
	)

	if err != nil {
		return "", err
	}

	var challenge challengeResponse

	err = json.Unmarshal(
		challengeRaw,
		&challenge,
	)

	if err != nil {
		return "", err
	}

	nonce, err := base64.RawURLEncoding.DecodeString(
		challenge.Nonce,
	)

	if err != nil {
		return "", err
	}

	// 3. sign

	domain := "BESHENCE-BANK-GATEWAY-PASS-CHALLENGE-V1"

	message := make([]byte, 0, len(domain)+len(nonce))

	message = append(
		message,
		[]byte(domain)...,
	)

	message = append(
		message,
		nonce...,
	)

	signature, err := slhdsa.SignRandomized(
		&privateKey,
		rand.Reader,
		slhdsa.NewMessage(message),
		nil,
	)

	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	// 4. JWT

	tokenRaw, err := post(
		gatewayURL+"/bank/"+bankID+"/challenge",
		map[string]string{
			"s": signatureB64,
		},
		"",
	)

	if err != nil {
		return "", err
	}

	var token tokenResponse

	err = json.Unmarshal(
		tokenRaw,
		&token,
	)

	if err != nil {
		return "", err
	}

	log.Println("[gateway] got token")

	return token.AccessToken, nil
}

func parseTokenExpiry(
	raw string,
) (time.Time, error) {

	parsed, _, err := new(jwt.Parser).
		ParseUnverified(
			raw,
			jwt.MapClaims{},
		)

	if err != nil {
		return time.Time{}, err
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)

	if !ok {
		return time.Time{}, errors.New("invalid claims")
	}

	exp, ok := claims["exp"].(float64)

	if !ok {
		return time.Time{}, errors.New("missing exp")
	}

	return time.Unix(
		int64(exp),
		0,
	), nil
}
