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
	bankDecapsulationKey           *mlkem.DecapsulationKey1024
	bankDecapsulationKeyErr        error
	bankEncapsulationKey           *mlkem.EncapsulationKey1024
	bankKeypairOnce                sync.Once
	bankEncapsulationKeyBase64     string
	bankEncapsulationKeyBase64Once sync.Once
	bankID                         string
	bankIDOnce                     sync.Once
)

func loadOrGenerateBankKeypair(db *gorm.DB) (*mlkem.DecapsulationKey1024, *mlkem.EncapsulationKey1024, error) {
	const keyName = "bank_decapsulation_key"
	var setting models.Setting
	var decapsulationKey *mlkem.DecapsulationKey1024

	err := db.First(&setting, "key = ?", keyName).Error

	switch {
	case err == nil:
		raw, err := base64.RawURLEncoding.DecodeString(setting.Value)
		if err != nil {
			return nil, nil, err
		}

		decapsulationKey, err = mlkem.NewDecapsulationKey1024(raw)
		if err != nil {
			return nil, nil, err
		}

	case errors.Is(err, gorm.ErrRecordNotFound):
		decapsulationKey, err = mlkem.GenerateKey1024()
		if err != nil {
			return nil, nil, err
		}

		err = db.Create(&models.Setting{
			Key:   keyName,
			Value: base64.RawURLEncoding.EncodeToString(decapsulationKey.Bytes()),
		}).Error

		if err != nil {
			return nil, nil, err
		}

	default:
		return nil, nil, err
	}

	return decapsulationKey, decapsulationKey.EncapsulationKey(), nil
}

func loadOrGenerateBankKeypairOnce(db *gorm.DB) {
	bankKeypairOnce.Do(func() {
		bankDecapsulationKey, bankEncapsulationKey, bankDecapsulationKeyErr = loadOrGenerateBankKeypair(db)
	})
}

func GetBankKeypair(db *gorm.DB) (*mlkem.DecapsulationKey1024, *mlkem.EncapsulationKey1024, error) {
	loadOrGenerateBankKeypairOnce(db)
	return bankDecapsulationKey, bankEncapsulationKey, bankDecapsulationKeyErr
}

func GetBankDecapsulationKey(db *gorm.DB) (*mlkem.DecapsulationKey1024, error) {
	loadOrGenerateBankKeypairOnce(db)
	return bankDecapsulationKey, bankDecapsulationKeyErr
}

func GetBankEncapsulationKey(db *gorm.DB) (*mlkem.EncapsulationKey1024, error) {
	loadOrGenerateBankKeypairOnce(db)
	return bankEncapsulationKey, bankDecapsulationKeyErr
}

func GetBankEncapsulationKeyBase64(db *gorm.DB) string {
	bankEncapsulationKeyBase64Once.Do(func() {
		key, _ := GetBankEncapsulationKey(db)
		bankEncapsulationKeyBase64 = base64.StdEncoding.EncodeToString(key.Bytes())
	})
	return bankEncapsulationKeyBase64
}

func generateBankID(key *mlkem.EncapsulationKey1024) string {
	h := sha3.New256()

	h.Write([]byte("BESHENCE-BANK-ID-V1"))
	h.Write(key.Bytes())

	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	encodedStr := encoder.EncodeToString(h.Sum(nil))
	return strings.ToLower(encodedStr)
}

func generateBankIDOnce(db *gorm.DB) {
	bankIDOnce.Do(func() {
		key, _ := GetBankEncapsulationKey(db)
		bankID = generateBankID(key)
	})
}

func GetBankID(db *gorm.DB) string {
	generateBankIDOnce(db)
	return bankID
}
