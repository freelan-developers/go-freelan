//go:generate stringer -type CipherSuite
//go:generate stringer -type EllipticCurve

package fscp

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

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
func (s CipherSuiteSlice) FindCommon(others CipherSuiteSlice) (CipherSuite, error) {
	for _, value := range s {
		for _, other := range others {
			if value == other {
				return value, nil
			}
		}
	}

	return NullCipherSuite, errors.New("no acceptable cipher suite could be found")
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
		SECT571K1,
		SECP384R1,
		SECP521R1,
	}
}

// FindCommon returns the first elliptic curve that is found in both slices.
func (s EllipticCurveSlice) FindCommon(others EllipticCurveSlice) (EllipticCurve, error) {
	for _, value := range s {
		for _, other := range others {
			if value == other {
				return value, nil
			}
		}
	}

	return 0, errors.New("no acceptable elliptic curve could be found")
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

// ClientSecurity contains all the security settings of a client.
type ClientSecurity struct {
	Certificate       *x509.Certificate
	PrivateKey        *rsa.PrivateKey
	PresharedKey      []byte
	RemoteCertificate *x509.Certificate
	CipherSuites      CipherSuiteSlice
	EllipticCurves    EllipticCurveSlice
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
func (s *ClientSecurity) Validate() error {
	if s.Certificate != nil {
		if s.PrivateKey == nil {
			return errors.New("a certificate was provided but not its associated private key")
		}
	} else if s.PresharedKey == nil {
		// If no certificate and no preshared key were set, we set the default preshared key.
		s.SetPresharedKeyFromPassphrase(DefaultPresharedKeyPassphrase, DefaultPresharedKeySalt, DefaultPresharedKeyIterations)
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

		return rsa.SignPSS(rand.Reader, s.PrivateKey, crypto.SHA256, hashed[:], nil)
	}

	hash := hmac.New(sha256.New, s.PresharedKey)
	hash.Write(cleartext)

	return hash.Sum(nil), nil
}

// Verify a signature.
func (s ClientSecurity) Verify(cleartext []byte, signature []byte) error {
	if s.RemoteCertificate != nil {
		hashed := sha256.Sum256(cleartext)

		return rsa.VerifyPSS(s.RemoteCertificate.PublicKey.(*rsa.PublicKey), crypto.SHA256, hashed[:], signature, nil)
	}

	hash := hmac.New(sha256.New, s.PresharedKey)
	hash.Write(cleartext)
	reference := hash.Sum(nil)

	if !hmac.Equal(reference, signature) {
		return fmt.Errorf("HMAC signature does not match: expected %s but got %s", hex.EncodeToString(reference), hex.EncodeToString(signature))
	}

	return nil
}
