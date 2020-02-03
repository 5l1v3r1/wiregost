// Wiregost - Golang Exploitation Framework
// Copyright © 2020 Para
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package certs

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/maxlandon/wiregost/server/db"
	"github.com/maxlandon/wiregost/server/log"
)

const (
	// RSAKeySize - Default size of RSA keys in bits
	RSAKeySize = 2048

	// Certs are valid for 3 years, minus up to 1 year from Now()
	validFor = 3 * (365 * 24 * time.Hour)

	// ECCKey - Namespace for ECC keys
	ECCKey = "ecc"

	// RSAKey - Namespace for RSA keys
	RSAKey = "rsa"
)

var (
	// Add logger here
	certsLog = log.ServerLogger("certs", "certificates")

	// ErrCertDoesNotExist - Returned if a GetCertificate() is called for cert/cn that does not exist
	ErrCertDoesNotExist = errors.New("Certificate does not exist")
)

// CertificateKeyPair - Single struct with KeyType/Cert/PrivateKey
type CertificateKeyPair struct {
	KeyType     string `json:"key_type"`
	Certificate []byte `json:"certificate"`
	PrivateKey  []byte `json:"private_key"`
}

// GetCertificate - Get the PEM encoded certificate & key for a host
func GetCertificate(caType string, keyType string, commonName string) ([]byte, []byte, error) {

	if keyType != ECCKey && keyType != RSAKey {
		return nil, nil, fmt.Errorf("Invalid key type '%s'", keyType)
	}

	certsLog.Infof("Getting certificate ca type = %s, cn = '%s'", caType, commonName)
	bucket, err := db.GetBucket(caType)
	if err != nil {
		return nil, nil, err
	}
	rawKeyPair, err := bucket.Get(fmt.Sprintf("%s_%s", keyType, commonName))
	if err == badger.ErrKeyNotFound {
		return nil, nil, ErrCertDoesNotExist
	}
	if err != nil {
		return nil, nil, err
	}
	keyPair := &CertificateKeyPair{}
	err = json.Unmarshal(rawKeyPair, keyPair)
	if err != nil {
		return nil, nil, err
	}
	return keyPair.Certificate, keyPair.PrivateKey, nil
}

// GetECCCertificate - Get an ECC certificate
func GetECCCertificate(caType string, commonName string) ([]byte, []byte, error) {
	return GetCertificate(caType, commonName, ECCKey)
}

// GetRSACertificate - Get a RSA certificate
func GetRSACertificate(caType string, commonName string) ([]byte, []byte, error) {
	return GetCertificate(caType, commonName, RSAKey)
}

// SaveCertificate - Save the certificate and the key in the file system
func SaveCertificate(caType string, keyType string, commonName string, cert []byte, key []byte) error {

	if keyType != ECCKey && keyType != RSAKey {
		return fmt.Errorf("Invalid key type '%s'", keyType)
	}

	bucket, err := db.GetBucket(caType)
	if err != nil {
		return err
	}
	bucket.Log.Infof("Saving certificate for cn = '%s'", commonName)
	keyPair, err := json.Marshal(CertificateKeyPair{
		KeyType:     keyType,
		Certificate: cert,
		PrivateKey:  key,
	})
	if err != nil {
		bucket.Log.Errorf("Failed to marshal key pair %s", err)
		return err
	}
	return bucket.Set(fmt.Sprintf("%s_%s", keyType, commonName), keyPair)
}

// RemoveCertificate - Remove a certificate from the cert store
func RemoveCertificate(caType string, keyType string, commonName string) error {

	if keyType != ECCKey && keyType != RSAKey {
		return fmt.Errorf("Invalid key type '%s'", keyType)
	}

	bucket, err := db.GetBucket(caType)
	if err != nil {
		return err
	}

	return bucket.Delete(fmt.Sprintf("%s_%s", keyType, commonName))
}

// [ Generic Certificate Functions ] -------------------------------------------------------------------------//

// GenerateECCCertificate - Generate a TLS certificate with the given parameters.
func GenerateECCCertificate(caType string, commonName string, isCA bool, isClient bool) ([]byte, []byte) {

	certsLog.Infof("Generating TLS certificate (ECC) for '%s' ...", commonName)

	var privateKey interface{}
	var err error

	// Generate private key
	privateKey, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		certsLog.Fatalf("Failed to generate private key: %s", err)
	}

	return generateCertificate(caType, commonName, isCA, isClient, privateKey)
}

// GenerateRSACertificate - Generates a 2048 RSA certificate
func GenerateRSACertificate(caType string, commonName string, isCA bool, isClient bool) ([]byte, []byte) {

	certsLog.Infof("Generating TLS certificate (RSA) for '%s' ...", commonName)

	var privateKey interface{}
	var err error

	// Generate private key
	privateKey, err = rsa.GenerateKey(rand.Reader, RSAKeySize)
	if err != nil {
		certsLog.Fatalf("Failed to generate private key %s", err)
	}
	return generateCertificate(caType, commonName, isCA, isClient, privateKey)
}

func generateCertificate(caType string, commonName string, isCA bool, isClient bool, privateKey interface{}) ([]byte, []byte) {

	// Valid times, subtract random days from .Now()
	notBefore := time.Now()
	days := randomInt(365) * -1 // Within -1 year
	notBefore = notBefore.AddDate(0, 0, days)
	notAfter := notBefore.Add(validFor)
	certsLog.Infof("Valid from %v to %v", notBefore, notAfter)

	// Serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	certsLog.Infof("Serial Number: %d", serialNumber)

	var extKeyUsage []x509.ExtKeyUsage

	if isCA {
		certsLog.Infof("Authority certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageAny}
	} else if isClient {
		certsLog.Infof("Client authentication certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		certsLog.Infof("Server authentication certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	certsLog.Infof("ExtKeyUsage = %v", extKeyUsage)

	// Certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{""},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: isCA,
	}

	if !isClient {
		// Host or IP address
		if ip := net.ParseIP(commonName); ip != nil {
			certsLog.Infof("Certificate authenticates IP address: %v", ip)
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			certsLog.Infof("Certificate authenticates host: %v", commonName)
			template.DNSNames = append(template.DNSNames, commonName)
		}
	} else {
		certsLog.Infof("Client certificate authenticates CN: %v", commonName)
		template.Subject.CommonName = commonName
	}

	// Sign certificate or self-sign if CA
	var err error
	var derBytes []byte
	if isCA {
		certsLog.Infof("Ceritificate is an AUTHORITY")
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, publicKey(privateKey), privateKey)
	} else {
		caCert, caKey, err := GetCertificateAuthority(caType) // Sign the new ceritificate with our CA
		if err != nil {
			certsLog.Fatalf("Invalid ca type (%s): %v", caType, err)
		}
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, caCert, publicKey(privateKey), caKey)
	}
	if err != nil {
		// We maybe don't want this to be fatal, but it should basically never happen afaik
		certsLog.Fatalf("Failed to create certificate: %s", err)
	}

	// Encode certificate and key
	certOut := bytes.NewBuffer([]byte{})
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyOut := bytes.NewBuffer([]byte{})
	pem.Encode(keyOut, pemBlockForKey(privateKey))

	return certOut.Bytes(), keyOut.Bytes()
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch key := priv.(type) {
	case *rsa.PrivateKey:
		data := x509.MarshalPKCS1PrivateKey(key)
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: data}
	case *ecdsa.PrivateKey:
		data, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			certsLog.Fatalf("Unable to marshal ECDSA private key: %v", err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: data}
	default:
		return nil
	}
}

func randomInt(max int) int {
	buf := make([]byte, 4)
	rand.Read(buf)
	i := binary.LittleEndian.Uint32(buf)
	return int(i) % max
}
