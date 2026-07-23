package gateway

import (
	"bank/internal/settings"
	"bytes"
	"crypto/hmac"
	"crypto/sha3"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type challengeResponse struct {
	Ciphertext string `json:"ciphertext"`
}

type tokenResponse struct {
	AccessToken string `json:"token"`
}

func StartPublisher(db *gorm.DB) {
	interval := 15 * time.Minute // TODO: from env

	go func() {
		publish(db)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			publish(db)
		}
	}()
}

func publish(db *gorm.DB) {
	err := publishOnce(db)

	if err != nil {
		log.Println("[gateway] publish failed:", err)
		return
	}

	log.Println("[gateway] published successfully")
}

func publishOnce(db *gorm.DB) error {
	gatewayURL := "https://gateway.beshence.com/api" // TODO: from env

	bankID := settings.GetBankID(db)

	_, ek, err := settings.GetBankKeypair(db)

	if err != nil {
		return err
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
		return err
	}

	// 2. get challenge

	challengeRaw, err := get(
		gatewayURL+"/bank/"+bankID+"/challenge",
		"",
	)

	if err != nil {
		return err
	}

	var challenge challengeResponse

	err = json.Unmarshal(
		challengeRaw,
		&challenge,
	)

	if err != nil {
		return err
	}

	ciphertext, err := base64.RawURLEncoding.DecodeString(
		challenge.Ciphertext,
	)

	if err != nil {
		return err
	}

	// 3. decapsulate

	dk, err := settings.GetBankDecapsulationKey(db)

	if err != nil {
		return err
	}

	sharedSecret, err := dk.Decapsulate(ciphertext)

	if err != nil {
		return err
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
		return err
	}

	var token tokenResponse

	err = json.Unmarshal(
		tokenRaw,
		&token,
	)

	if err != nil {
		return err
	}

	// 6. publish URLs

	_, err = post(
		gatewayURL+"/bank/"+bankID+"/urls",
		map[string][]string{
			"api_urls": settings.GetAPIUrls(),
		},
		token.AccessToken,
	)

	return err
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

func post(url string, body any, token string) ([]byte, error) {

	data, err := json.Marshal(body)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		url,
		bytes.NewReader(data),
	)

	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	if token != "" {
		req.Header.Set(
			"Authorization",
			"Bearer "+token,
		)
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"gateway returned %s: %s",
			resp.Status,
			string(result),
		)
	}

	return result, nil
}

func get(url string, token string) ([]byte, error) {

	req, err := http.NewRequest(
		"GET",
		url,
		nil,
	)

	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set(
			"Authorization",
			"Bearer "+token,
		)
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"gateway returned %s: %s",
			resp.Status,
			string(result),
		)
	}

	return result, nil
}
