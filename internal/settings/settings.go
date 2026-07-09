package settings

import (
	"bank/internal/database/models"
	"crypto/mlkem"
	"crypto/sha3"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"strings"
	"sync"

	"gorm.io/gorm"
)

var (
	bankDecapsulationKey     *mlkem.DecapsulationKey1024
	bankDecapsulationKeyErr  error
	bankDecapsulationKeyOnce sync.Once
	bankEncapsulationKey     *mlkem.EncapsulationKey1024
	bankEncapsulationKeyErr  error
	bankEncapsulationKeyOnce sync.Once
	bankID                   string
	bankIDOnce               sync.Once
)

func loadOrGenerateBankDecapsulationKey(db *gorm.DB) (*mlkem.DecapsulationKey1024, error) {
	const keyName = "bank_decapsulation_key"
	var setting models.Setting
	var decapsulationKey *mlkem.DecapsulationKey1024

	err := db.First(&setting, "key = ?", keyName).Error

	switch {
	case err == nil:
		raw, err := base64.RawURLEncoding.DecodeString(setting.Value)
		if err != nil {
			return nil, err
		}

		decapsulationKey, err = mlkem.NewDecapsulationKey1024(raw)
		if err != nil {
			return nil, err
		}

	case errors.Is(err, gorm.ErrRecordNotFound):
		decapsulationKey, err = mlkem.GenerateKey1024()
		if err != nil {
			return nil, err
		}

		err = db.Create(&models.Setting{
			Key:   keyName,
			Value: base64.RawURLEncoding.EncodeToString(decapsulationKey.Bytes()),
		}).Error

		if err != nil {
			return nil, err
		}

	default:
		return nil, err
	}

	return decapsulationKey, nil
}

func loadOrGenerateBankEncapsulationKey(db *gorm.DB) (*mlkem.EncapsulationKey1024, error) {
	decapsulationKey, _ := loadOrGenerateBankDecapsulationKey(db)
	return decapsulationKey.EncapsulationKey(), nil
}

func generateBankID(key *mlkem.EncapsulationKey1024) string {
	h := sha3.New256()

	h.Write([]byte("BESHENCE-BANK-ID-v1"))
	h.Write(key.Bytes())

	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	encodedStr := encoder.EncodeToString(h.Sum(nil))
	return strings.ToLower(encodedStr)
}

func GetBankDecapsulationKey(db *gorm.DB) (*mlkem.DecapsulationKey1024, error) {
	bankDecapsulationKeyOnce.Do(func() {
		bankDecapsulationKey, bankDecapsulationKeyErr = loadOrGenerateBankDecapsulationKey(db)
	})
	return bankDecapsulationKey, bankDecapsulationKeyErr
}

func GetBankEncapsulationKey(db *gorm.DB) (*mlkem.EncapsulationKey1024, error) {
	bankEncapsulationKeyOnce.Do(func() {
		bankEncapsulationKey, bankEncapsulationKeyErr = loadOrGenerateBankEncapsulationKey(db)
	})
	return bankEncapsulationKey, bankEncapsulationKeyErr
}

func GetBankID(db *gorm.DB) string {
	bankIDOnce.Do(func() {
		key, _ := loadOrGenerateBankEncapsulationKey(db)
		bankID = generateBankID(key)
	})

	return bankID
}
