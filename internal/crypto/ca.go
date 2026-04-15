package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

type CA struct {
	certDir string
	caCert  *x509.Certificate
	caKey   *ecdsa.PrivateKey
}

func NewCA(certDir string) (*CA, error) {
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return nil, err
	}

	ca := &CA{certDir: certDir}
	err := ca.loadOrGenerateCA()
	if err != nil {
		return nil, err
	}

	return ca, nil
}

func (c *CA) loadOrGenerateCA() error {
	certPath := filepath.Join(c.certDir, "ca.crt")
	keyPath := filepath.Join(c.certDir, "ca.key")

	if _, err := os.Stat(certPath); err == nil {
		// Load existing
		certBytes, err := os.ReadFile(certPath)
		if err != nil {
			return err
		}
		keyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return err
		}

		block, _ := pem.Decode(certBytes)
		c.caCert, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return err
		}

		block, _ = pem.Decode(keyBytes)
		c.caKey, err = x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return err
		}
		return nil
	}

	// Generate new
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "Chat Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return err
	}

	c.caCert = template
	c.caKey = key

	// Save to disk
	certFile, _ := os.Create(certPath)
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certFile.Close()

	keyFile, _ := os.Create(keyPath)
	keyBytes, _ := x509.MarshalECPrivateKey(key)
	pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	keyFile.Close()

	return nil
}

func (c *CA) IssueUserCert(userID string, dnsNames []string) ([]byte, []byte, error) {
	userKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: userID,
		},
		DNSNames:    dnsNames,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, c.caCert, &userKey.PublicKey, c.caKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, _ := x509.MarshalECPrivateKey(userKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	return certPEM, keyPEM, nil
}

func (c *CA) GetCACert() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.caCert.Raw})
}
