package settings

import (
	"bank/internal/database/models"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base32"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/cloudflare/circl/sign/slhdsa"
	"gorm.io/gorm"
)

var (
	bankRootPrivateKey          slhdsa.PrivateKey
	bankRootPrivateKeyErr       error
	bankRootPublicKey           slhdsa.PublicKey
	bankKeypairOnce             sync.Once
	bankRootPublicKeyBase64     string
	bankRootPublicKeyBase64Once sync.Once
	bankID                      string
	bankIDOnce                  sync.Once

	bankCAKey     *ecdsa.PrivateKey
	bankCACert    *x509.Certificate
	bankCACertPEM string
	bankCAKeyPEM  string
	bankCAOnce    sync.Once
	bankCAErr     error
)

const (
	SettingPrivateKey = "bank_private_key"
	SettingCAKey      = "bank_tls_ca_private_key"
	SettingCACert     = "bank_tls_ca_certificate"
)

func loadOrGenerateBankRootKeypair(db *gorm.DB) (slhdsa.PrivateKey, slhdsa.PublicKey, error) {
	var setting models.Setting
	var privateKey slhdsa.PrivateKey

	err := db.First(&setting, "key = ?", SettingPrivateKey).Error

	switch {
	case err == nil:
		raw, err := base64.RawURLEncoding.DecodeString(setting.Value)
		if err != nil {
			return slhdsa.PrivateKey{}, slhdsa.PublicKey{}, err
		}

		privateKey = slhdsa.PrivateKey{ID: slhdsa.SHAKE_256s}
		err = privateKey.UnmarshalBinary(raw)
		if err != nil {
			return slhdsa.PrivateKey{}, slhdsa.PublicKey{}, err
		}
		break
	case errors.Is(err, gorm.ErrRecordNotFound):
		_, privateKey, err = slhdsa.GenerateKey(
			rand.Reader,
			slhdsa.SHAKE_256s,
		)
		if err != nil {
			return slhdsa.PrivateKey{}, slhdsa.PublicKey{}, err
		}

		privateKeyBytes, err := privateKey.MarshalBinary()
		if err != nil {
			return slhdsa.PrivateKey{}, slhdsa.PublicKey{}, err
		}

		err = db.Create(&models.Setting{
			Key:   SettingPrivateKey,
			Value: base64.RawURLEncoding.EncodeToString(privateKeyBytes),
		}).Error

		if err != nil {
			return slhdsa.PrivateKey{}, slhdsa.PublicKey{}, err
		}
		break
	default:
		if err != nil {
			return slhdsa.PrivateKey{}, slhdsa.PublicKey{}, err
		}
		break
	}

	return privateKey, privateKey.PublicKey(), nil
}

func loadOrGenerateBankRootKeypairOnce(db *gorm.DB) {
	bankKeypairOnce.Do(func() {
		bankRootPrivateKey, bankRootPublicKey, bankRootPrivateKeyErr = loadOrGenerateBankRootKeypair(db)
	})
}

func GetBankRootKeypair(db *gorm.DB) (slhdsa.PrivateKey, slhdsa.PublicKey, error) {
	loadOrGenerateBankRootKeypairOnce(db)
	return bankRootPrivateKey, bankRootPublicKey, bankRootPrivateKeyErr
}

func GetBankRootPrivateKey(db *gorm.DB) (slhdsa.PrivateKey, error) {
	loadOrGenerateBankRootKeypairOnce(db)
	return bankRootPrivateKey, bankRootPrivateKeyErr
}

func GetBankRootPublicKey(db *gorm.DB) (slhdsa.PublicKey, error) {
	loadOrGenerateBankRootKeypairOnce(db)
	return bankRootPublicKey, bankRootPrivateKeyErr
}

func GetBankRootPublicKeyBase64(db *gorm.DB) string {
	bankRootPublicKeyBase64Once.Do(func() {
		key, _ := GetBankRootPublicKey(db)
		keyBytes, _ := key.MarshalBinary()
		bankRootPublicKeyBase64 = base64.RawURLEncoding.EncodeToString(keyBytes)
	})
	return bankRootPublicKeyBase64
}

func getBankID(key slhdsa.PublicKey) string {
	h := sha256.New()

	keyBytes, _ := key.MarshalBinary()

	h.Write([]byte("BESHENCE-BANK-ID-V1"))
	h.Write(keyBytes)

	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	encodedStr := encoder.EncodeToString(h.Sum(nil))
	return strings.ToLower(encodedStr)
}

func getBankIDOnce(db *gorm.DB) {
	bankIDOnce.Do(func() {
		key, _ := GetBankRootPublicKey(db)
		bankID = getBankID(key)
	})
}

func GetBankID(db *gorm.DB) string {
	getBankIDOnce(db)
	return bankID
}

func loadOrGenerateCA(db *gorm.DB) (*ecdsa.PrivateKey, *x509.Certificate, error) {
	var keySetting models.Setting
	var certSetting models.Setting

	keyErr := db.First(&keySetting, "key = ?", SettingCAKey).Error
	certErr := db.First(&certSetting, "key = ?", SettingCACert).Error

	if keyErr == nil && certErr == nil {
		keyBlock, _ := pem.Decode([]byte(keySetting.Value))
		if keyBlock == nil {
			return nil, nil, errors.New("invalid CA private key PEM")
		}

		key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, nil, err
		}

		certBlock, _ := pem.Decode([]byte(certSetting.Value))
		if certBlock == nil {
			return nil, nil, errors.New("invalid CA certificate PEM")
		}

		cert, err := x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			return nil, nil, err
		}

		if time.Now().After(cert.NotAfter.Add(-24*time.Hour)) || time.Now().Before(cert.NotBefore) {
			if err := db.Delete(
				&models.Setting{},
				"key IN (?, ?)",
				SettingCAKey,
				SettingCACert,
			).Error; err != nil {
				return nil, nil, err
			}
		} else {
			return key, cert, nil
		}
	}

	if !errors.Is(keyErr, gorm.ErrRecordNotFound) ||
		!errors.Is(certErr, gorm.ErrRecordNotFound) {

		return nil, nil, errors.New("failed loading CA settings")
	}

	key, err := ecdsa.GenerateKey(
		elliptic.P521(),
		rand.Reader,
	)

	if err != nil {
		return nil, nil, err
	}

	serial, err := rand.Int(
		rand.Reader,
		new(big.Int).Lsh(big.NewInt(1), 128),
	)

	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "Beshence Bank CA",
			Organization: []string{
				"Beshence",
			},
		},
		NotBefore: time.Now(),
		NotAfter: time.Now().AddDate(
			25,
			0,
			0,
		),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign |
			x509.KeyUsageCRLSign |
			x509.KeyUsageDigitalSignature,
	}

	der, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&key.PublicKey,
		key,
	)

	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(der)

	if err != nil {
		return nil, nil, err
	}

	keyBytes, err := x509.MarshalECPrivateKey(key)

	if err != nil {
		return nil, nil, err
	}

	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: keyBytes,
		},
	)

	certPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: der,
		},
	)

	err = db.Create(&models.Setting{
		Key:   SettingCAKey,
		Value: string(keyPEM),
	}).Error

	if err != nil {
		return nil, nil, err
	}

	err = db.Create(&models.Setting{
		Key:   SettingCACert,
		Value: string(certPEM),
	}).Error

	if err != nil {
		return nil, nil, err
	}

	return key, cert, nil
}

func loadOrGenerateCAOnce(db *gorm.DB) {
	bankCAOnce.Do(func() {

		bankCAKey, bankCACert, bankCAErr =
			loadOrGenerateCA(db)

		if bankCAErr != nil {
			return
		}

		keyBytes, _ := x509.MarshalECPrivateKey(bankCAKey)

		bankCAKeyPEM = string(
			pem.EncodeToMemory(
				&pem.Block{
					Type:  "EC PRIVATE KEY",
					Bytes: keyBytes,
				},
			),
		)

		certBytes := bankCACert.Raw

		bankCACertPEM = string(
			pem.EncodeToMemory(
				&pem.Block{
					Type:  "CERTIFICATE",
					Bytes: certBytes,
				},
			),
		)
	})
}

func GetBankCA(db *gorm.DB) (*ecdsa.PrivateKey, *x509.Certificate, error) {
	loadOrGenerateCAOnce(db)

	return bankCAKey, bankCACert, bankCAErr
}

func GetBankCACertPEM(db *gorm.DB) string {
	loadOrGenerateCAOnce(db)

	return bankCACertPEM
}

func GetBankCAKeyPEM(db *gorm.DB) string {
	loadOrGenerateCAOnce(db)

	return bankCAKeyPEM
}
