//go:generate stringer -type CipherSuite
//go:generate stringer -type EllipticCurve

package fscp

import (
	"crypto"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// CipherSuite represents a cipher suite.
type CipherSuite uint8

const (
	// NullCipherSuite represents an invalid cipher suite.
	NullCipherSuite CipherSuite = 0x00
	// ECDHERSAAES128GCMSHA256 is the ECDHE-RSA-AES-128-GCM-SHA256 cipher suite.
	ECDHERSAAES128GCMSHA256 CipherSuite = 0x01
	// ECDHERSAAES256GCMSHA384 is the ECDHE-RSA-AES-256-GCM-SHA384 cipher suite.
	ECDHERSAAES256GCMSHA384 CipherSuite = 0x02
)

// BlockSize returns the block size.
func (s CipherSuite) BlockSize() int {
	switch s {
	case NullCipherSuite:
		return 0
	case ECDHERSAAES128GCMSHA256:
		return 16
	case ECDHERSAAES256GCMSHA384:
		return 32
	}

	panic(fmt.Errorf("Unknown cipher suite: %s", s))

}

// CipherSuiteSlice represents a slice of cipher suites.
type CipherSuiteSlice []CipherSuite

// DefaultCipherSuites returns the default cipher suites.
func DefaultCipherSuites() CipherSuiteSlice {
	return CipherSuiteSlice{
		ECDHERSAAES256GCMSHA384,
		ECDHERSAAES128GCMSHA256,
	}
}

// FindCommon returns the first cipher suite that is found in both slices.
func (s CipherSuiteSlice) FindCommon(others CipherSuiteSlice) CipherSuite {
	for _, value := range s {
		for _, other := range others {
			if value == other {
				return value
			}
		}
	}

	return NullCipherSuite
}

func (s CipherSuiteSlice) String() string {
	var strs []string

	for _, value := range s {
		strs = append(strs, value.String())
	}

	return strings.Join(strs, ",")
}

// EllipticCurve represents an elliptic curve.
type EllipticCurve uint8

const (
	// NullEllipticCurve represents an invalid elliptic curve.
	NullEllipticCurve EllipticCurve = 0x00
	// SECT571K1 is the SECT571K1 elliptic curve.
	SECT571K1 EllipticCurve = 0x01
	// SECP384R1 is the SECP384R1 elliptic curve.
	SECP384R1 EllipticCurve = 0x02
	// SECP521R1 is the SECP521R1 elliptic curve.
	SECP521R1 EllipticCurve = 0x03
)

// EllipticCurveSlice represents a slice of elliptic curves.
type EllipticCurveSlice []EllipticCurve

// DefaultEllipticCurves returns the default elliptic curves.
func DefaultEllipticCurves() EllipticCurveSlice {
	return EllipticCurveSlice{
		SECP384R1,
		SECP521R1,
	}
}

// Curve returns the associated elliptic curve.
func (c EllipticCurve) Curve() elliptic.Curve {
	switch c {
	case SECP384R1:
		return elliptic.P384()
	case SECP521R1:
		return elliptic.P521()
	default:
		return nil
	}
}

// FindCommon returns the first elliptic curve that is found in both slices.
func (s EllipticCurveSlice) FindCommon(others EllipticCurveSlice) EllipticCurve {
	for _, value := range s {
		for _, other := range others {
			if value == other {
				return value
			}
		}
	}

	return NullEllipticCurve
}

func (s EllipticCurveSlice) String() string {
	var strs []string

	for _, value := range s {
		strs = append(strs, value.String())
	}

	return strings.Join(strs, ",")
}

// A Signer signs data.
type Signer interface {
	Sign(cleartext []byte) ([]byte, error)
}

// A Verifier verifies signed data.
type Verifier interface {
	Verify(cleartext []byte, signature []byte) error
}

// GenerateLocalCertificate generates a default local X509 certificate for the
// current host.
func GenerateLocalCertificate() (*rsa.PrivateKey, *x509.Certificate, error) {
	hostname, err := os.Hostname()

	if err != nil {
		return nil, nil, fmt.Errorf("could not determine local hostname: %s", err)
	}

	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1653),
		Subject: pkix.Name{
			CommonName: hostname,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA private key: %s", err)
	}

	pub := &priv.PublicKey

	raw, err := x509.CreateCertificate(rand.Reader, ca, ca, pub, priv)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create X509 certificate: %s", err)
	}

	cert, err := x509.ParseCertificate(raw)

	return priv, cert, err
}

// ClientSecurity contains all the security settings of a client.
type ClientSecurity struct {
	Certificate    *x509.Certificate
	PrivateKey     *rsa.PrivateKey
	PresharedKey   []byte
	CipherSuites   CipherSuiteSlice
	EllipticCurves EllipticCurveSlice

	RemoteClientSecurity *RemoteClientSecurity
}

// RemoteClientSecurity represents the remote client security.
type RemoteClientSecurity struct {
	Certificate *x509.Certificate
}

// DefaultPresharedKeyPassphrase is the default preshared key passphrase.
const DefaultPresharedKeyPassphrase = ""

// DefaultPresharedKeySalt is the default preshared key salt.
var DefaultPresharedKeySalt = []byte("freelan")

// DefaultPresharedKeyIterations is the default preshared key iterations.
const DefaultPresharedKeyIterations = 2000

// SetPresharedKeyFromPassphrase set the preshared key from a passphrase and salt/iterations parameters.
func (s *ClientSecurity) SetPresharedKeyFromPassphrase(passphrase string, salt []byte, iterations int) {
	s.PresharedKey = pbkdf2.Key([]byte(passphrase), salt, iterations, sha256.Size, sha256.New)
}

// Validate the security.
func (s *ClientSecurity) Validate() (err error) {
	if s.Certificate != nil {
		if s.PrivateKey == nil {
			return errors.New("a certificate was provided but not its associated private key")
		}
	} else if s.PresharedKey == nil {
		// If no certificate and no preshared key were set, we generate a temporary certificate.
		if s.PrivateKey, s.Certificate, err = GenerateLocalCertificate(); err != nil {
			return
		}
	}

	if len(s.supportedCipherSuites()) == 0 {
		return errors.New("a least one cipher suite must be supported")
	}

	if len(s.supportedEllipticCurves()) == 0 {
		return errors.New("a least one elliptic curve must be supported")
	}

	return nil
}

func (s *ClientSecurity) supportedCipherSuites() CipherSuiteSlice {
	if s.CipherSuites == nil {
		return DefaultCipherSuites()
	}

	return s.CipherSuites
}

func (s *ClientSecurity) supportedEllipticCurves() EllipticCurveSlice {
	if s.EllipticCurves == nil {
		return DefaultEllipticCurves()
	}

	return s.EllipticCurves
}

// Sign a message.
func (s ClientSecurity) Sign(cleartext []byte) ([]byte, error) {
	if s.PrivateKey != nil {
		hashed := sha256.Sum256(cleartext)

		// This is necessary for interoperability with the legacy freelan
		// implementation.
		options := &rsa.PSSOptions{
			SaltLength: sha256.Size,
		}

		return rsa.SignPSS(rand.Reader, s.PrivateKey, crypto.SHA256, hashed[:], options)
	}

	hash := hmac.New(sha256.New, s.PresharedKey)
	hash.Write(cleartext)

	return hash.Sum(nil), nil
}

// Verify a signature.
func (s ClientSecurity) Verify(cleartext []byte, signature []byte) error {
	if s.RemoteClientSecurity == nil {
		return errors.New("cannot verify signature as no remote client security information is available")
	}

	if s.RemoteClientSecurity.Certificate != nil {
		hashed := sha256.Sum256(cleartext)

		return rsa.VerifyPSS(s.RemoteClientSecurity.Certificate.PublicKey.(*rsa.PublicKey), crypto.SHA256, hashed[:], signature, nil)
	}

	hash := hmac.New(sha256.New, s.PresharedKey)
	hash.Write(cleartext)
	reference := hash.Sum(nil)

	if !hmac.Equal(reference, signature) {
		return fmt.Errorf("HMAC signature does not match: expected %s but got %s", hex.EncodeToString(reference), hex.EncodeToString(signature))
	}

	return nil
}
