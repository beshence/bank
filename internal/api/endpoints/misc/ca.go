package misc

import (
	"bank/internal/api"
	"bank/internal/settings"
	"os"

	"crypto/hmac"
	"crypto/sha3"
	"encoding/base64"
	hashlib "hash"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CAV1(deps *api.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		disableTls := os.Getenv("BANK_DISABLE_TLS")
		disableTlsBool := false
		if disableTls == "true" || disableTls == "1" {
			disableTlsBool = true
		}

		if disableTlsBool {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err": "UNKNOWN",
			})
			return
		}

		if deps == nil || deps.DB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err": "UNKNOWN",
			})
			return
		}

		var body struct {
			Ciphertext string `json:"ciphertext"`
		}

		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err": "INVALID_JSON",
			})
			return
		}

		ciphertext, err := base64.RawURLEncoding.DecodeString(
			body.Ciphertext,
		)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err": "INVALID_CIPHERTEXT",
			})
			return
		}

		dk, err := settings.GetBankDecapsulationKey(deps.DB)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"err": "NO_KEY",
			})
			return
		}

		sharedSecret, err := dk.Decapsulate(ciphertext)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err": "INVALID_KEM",
			})
			return
		}

		caPEM := settings.GetBankCACertPEM(deps.DB)

		hash := sha3.Sum256(
			[]byte(caPEM),
		)

		mac := hmac.New(
			func() hashlib.Hash {
				return sha3.New256()
			},
			sharedSecret,
		)

		mac.Write([]byte(
			"BESHENCE_BANK_CA_V1",
		))

		mac.Write(hash[:])

		proof := mac.Sum(nil)

		c.JSON(http.StatusOK, gin.H{
			"err": "0",
			"ca":  caPEM,
			"proof": base64.RawURLEncoding.EncodeToString(
				proof,
			),
		})
	}
}
