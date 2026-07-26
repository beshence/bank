package tls

import (
	"bank/internal/settings"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"

	"math/big"
	"net"
	"net/url"
	"sync"
	"time"

	"gorm.io/gorm"
)

var (
	currentCertificate *tls.Certificate
	certificateMutex   sync.RWMutex
)

func GetTLSConfig(db *gorm.DB) (*tls.Config, error) {
	_, err := generateCertificate(db)

	if err != nil {
		return nil, err
	}

	return &tls.Config{
		GetCertificate: func(
			hello *tls.ClientHelloInfo,
		) (*tls.Certificate, error) {
			return getCertificate(db)
		},
		MinVersion: tls.VersionTLS12,
	}, nil
}

func getCertificate(db *gorm.DB) (*tls.Certificate, error) {
	certificateMutex.RLock()

	cert := currentCertificate

	certificateMutex.RUnlock()

	if cert != nil {
		leaf, err := x509.ParseCertificate(
			cert.Certificate[0],
		)

		if err == nil &&
			time.Now().Before(leaf.NotAfter.Add(-time.Hour)) {
			return cert, nil
		}
	}

	return generateCertificate(db)
}

func generateCertificate(db *gorm.DB) (*tls.Certificate, error) {
	certificateMutex.Lock()
	defer certificateMutex.Unlock()

	caKey, caCert, err := settings.GetBankCA(db)

	if err != nil {
		return nil, err
	}

	apiUrls := settings.GetAPIUrls()

	var dnsNames []string
	var ips []net.IP

	for _, raw := range apiUrls {
		u, err := url.Parse(raw)

		if err != nil {
			continue
		}

		host := u.Hostname()

		if host == "" {
			continue
		}

		if ip := net.ParseIP(host); ip != nil {
			ips = append(ips, ip)
		} else {
			dnsNames = append(dnsNames, host)
		}
	}

	key, err := ecdsa.GenerateKey(
		elliptic.P384(),
		rand.Reader,
	)

	if err != nil {
		return nil, err
	}

	serial, err := rand.Int(
		rand.Reader,
		new(big.Int).Lsh(big.NewInt(1), 128),
	)

	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "Beshence Bank " +
				settings.GetBankID(db),
		},
		NotBefore:   time.Now().Add(-time.Minute),
		NotAfter:    time.Now().AddDate(0, 0, 1),
		DNSNames:    dnsNames,
		IPAddresses: ips,
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	der, err := x509.CreateCertificate(
		rand.Reader,
		template,
		caCert,
		&key.PublicKey,
		caKey,
	)

	if err != nil {
		return nil, err
	}

	result := &tls.Certificate{
		Certificate: [][]byte{
			der,
			caCert.Raw,
		},
		PrivateKey: key,
	}

	currentCertificate = result

	return result, nil
}
