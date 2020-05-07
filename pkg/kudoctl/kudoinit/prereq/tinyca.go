package prereq

// This is a slightly modified version of controller-runtime/pkg/internal/testing/integration/internal/tinyca.go
// package which is sadly internal and can't be used directly. All the methods here are supposed to be FOR TESTING ONLY.
// This package is used to provide self-signed CA along with a CA signed server certificate (and key) for services running
// inside the cluster. This is IN NO WAY a generic certificate generation solution as it is tailored towards testing and demos.
// Generated server certificate is valid 1 week which is generous enough for testing and demos.
// More information: https://kubernetes.io/blog/2019/03/21/a-guide-to-kubernetes-admission-controllers/

import (
	"crypto"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	certutil "k8s.io/client-go/util/cert"
)

var (
	rsaKeySize = 2048 // a decent number, as of 2019
	bigOne     = big.NewInt(1)
)

// CertPair is a private key and certificate for use for client auth, as a CA, or serving.
type CertPair struct {
	Key  crypto.Signer
	Cert *x509.Certificate
}

// CertBytes returns the PEM-encoded version of the certificate for this pair.
func (k CertPair) CertBytes() []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: k.Cert.Raw,
	})
}

// AsBytes encodes key-pair in the appropriate formats for on-disk storage (PEM and PKCS8, respectively).
func (k CertPair) AsBytes() (cert []byte, key []byte, err error) {
	cert = k.CertBytes()

	rawKeyData, err := x509.MarshalPKCS8PrivateKey(k.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to encode private key: %v", err)
	}

	key = pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: rawKeyData,
	})

	return cert, key, nil
}

// TinyCA supports signing serving certs and client-certs for services
// and can be used as an auth mechanism with tests.
type TinyCA struct {
	CA         CertPair
	CN         string
	Service    string
	Namespace  string
	nextSerial *big.Int
}

// newPrivateKey generates a new private key of a relatively sane size (see rsaKeySize)
func newPrivateKey() (crypto.Signer, error) {
	return rsa.GenerateKey(crand.Reader, rsaKeySize)
}

func NewTinyCA(svc, ns string) (*TinyCA, error) {
	caPrivateKey, err := newPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("unable to generate private key for CA: %v", err)
	}
	cn := fmt.Sprintf("%s.%s.svc", svc, ns)
	caCfg := certutil.Config{CommonName: cn}
	caCert, err := certutil.NewSelfSignedCACert(caCfg, caPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to generate certificate for CA: %v", err)
	}

	return &TinyCA{
		CA:         CertPair{Key: caPrivateKey, Cert: caCert},
		CN:         cn,
		Service:    svc,
		Namespace:  ns,
		nextSerial: big.NewInt(1),
	}, nil
}

func (ca *TinyCA) makeCert(cfg certutil.Config) (CertPair, error) {
	now := time.Now()

	key, err := newPrivateKey()
	if err != nil {
		return CertPair{}, fmt.Errorf("unable to create private key: %v", err)
	}

	serial := new(big.Int).Set(ca.nextSerial)
	ca.nextSerial.Add(ca.nextSerial, bigOne)

	template := x509.Certificate{
		Subject:      pkix.Name{CommonName: cfg.CommonName, Organization: cfg.Organization},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,

		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: cfg.Usages,

		// technically not necessary for testing, but let's set anyway just in case.
		NotBefore: now.UTC(),
		// 1 week is just long enough for a long-term test, but not too long that anyone would
		// try to use this seriously.
		NotAfter: now.Add(168 * time.Hour).UTC(),
	}

	certRaw, err := x509.CreateCertificate(crand.Reader, &template, ca.CA.Cert, key.Public(), ca.CA.Key)
	if err != nil {
		return CertPair{}, fmt.Errorf("unable to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certRaw)
	if err != nil {
		return CertPair{}, fmt.Errorf("generated invalid certificate, could not parse: %v", err)
	}

	return CertPair{
		Key:  key,
		Cert: cert,
	}, nil
}

// NewServingCert returns a new CertPair for a serving HTTPS for a service. DNSNames are generated from the passed
// service and namespace
func (ca *TinyCA) NewServingCert() (CertPair, error) {
	return ca.makeCert(certutil.Config{
		CommonName: ca.CN,
		AltNames: certutil.AltNames{
			DNSNames: []string{
				ca.Service,
				fmt.Sprintf("%s.%s", ca.Service, ca.Namespace),
				fmt.Sprintf("%s.%s.svc", ca.Service, ca.Namespace),
			},
		},
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
}
