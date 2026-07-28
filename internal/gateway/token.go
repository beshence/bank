package gateway

import (
	"bank/internal/settings"
	"crypto/hmac"
	"crypto/sha3"
	"encoding/base64"
	"encoding/json"
	"errors"
	"hash"
	"log"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

var (
	tokenMu sync.Mutex

	cachedToken string
	tokenExpiry time.Time
)

type challengeResponse struct {
	Ciphertext string `json:"ciphertext"`
}

type tokenResponse struct {
	AccessToken string `json:"token"`
}

func GetGatewayToken(db *gorm.DB, gatewayURL string, bankID string) (string, error) {
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
	_, ek, err := settings.GetBankKeypair(db)

	if err != nil {
		return "", err
	}

	// 1. publish EK

	ekBase64 := base64.RawURLEncoding.EncodeToString(
		ek.Bytes(),
	)

	_, err = post(
		gatewayURL+"/bank/"+bankID+"/ek",
		map[string]string{
			"bankId": bankID,
			"ek":     ekBase64,
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

	ciphertext, err := base64.RawURLEncoding.DecodeString(
		challenge.Ciphertext,
	)

	if err != nil {
		return "", err
	}

	// 3. decapsulate

	dk, err := settings.GetBankDecapsulationKey(db)

	if err != nil {
		return "", err
	}

	sharedSecret, err := dk.Decapsulate(ciphertext)

	if err != nil {
		return "", err
	}

	// 4. proof

	proof := makeProof(
		sharedSecret,
		ciphertext,
	)

	// 5. JWT

	tokenRaw, err := post(
		gatewayURL+"/bank/"+bankID+"/challenge",
		map[string]string{
			"proof": base64.RawURLEncoding.EncodeToString(proof),
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

func makeProof(key []byte, data []byte) []byte {
	h := hmac.New(
		func() hash.Hash {
			return sha3.New256()
		},
		key,
	)

	h.Write(data)

	return h.Sum(nil)
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
