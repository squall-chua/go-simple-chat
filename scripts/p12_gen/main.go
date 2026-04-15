package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"software.sslmate.com/src/go-pkcs12"
)

func main() {
	certFile := "testuser.crt"
	keyFile := "testuser.key"
	caFile := "certs/ca.crt"
	outputFile := "web-demo-import-me.p12"
	password := "password"

	// 1. Read files
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		log.Fatalf("failed to read cert: %v", err)
	}
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		log.Fatalf("failed to read key: %v", err)
	}
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		log.Fatalf("failed to read ca: %v", err)
	}

	// 2. Decode PEM
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		log.Fatal("failed to decode certificate")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		log.Fatalf("failed to parse certificate: %v", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		log.Fatal("failed to decode key")
	}
	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Fallback to PCKS1
		key, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if err != nil {
			log.Fatalf("failed to parse private key: %v", err)
		}
	}

	caBlock, _ := pem.Decode(caPEM)
	if caBlock == nil {
		log.Fatal("failed to decode ca cert")
	}
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		log.Fatalf("failed to parse ca certificate: %v", err)
	}

	// 3. Encode to PKCS12
	p12Data, err := pkcs12.Modern.Encode(key, cert, []*x509.Certificate{caCert}, password)
	if err != nil {
		log.Fatalf("failed to encode pkcs12: %v", err)
	}

	// 4. Write to file
	if err := os.WriteFile(outputFile, p12Data, 0644); err != nil {
		log.Fatalf("failed to write output: %v", err)
	}

	fmt.Printf("Successfully created %s\nPassword is: %s\nImport this file into your browser to enable mTLS.\n", outputFile, password)
}
