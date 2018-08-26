package fscp

import (
	"crypto/rsa"
	"crypto/x509"
	"errors"
)

// CipherSuite represents a cipher suite.
type CipherSuite uint8

const (
	// CipherSuiteECDHERSAAES128GCMSHA256 is the ECDHE-RSA-AES-128-GCM-SHA256 cipher suite.
	CipherSuiteECDHERSAAES128GCMSHA256 = 0x01
	// CipherSuiteECDHERSAAES256GCMSHA384 is the ECDHE-RSA-AES-256-GCM-SHA384 cipher suite.
	CipherSuiteECDHERSAAES256GCMSHA384 = 0x02
)

// EllipticCurve represents an elliptic curve.
type EllipticCurve uint8

const (
	// EllipticCurveSECT571K1 is the SECT571K1 elliptic curve.
	EllipticCurveSECT571K1 = 0x01
	// EllipticCurveSECP384R1 is the SECP384R1 elliptic curve.
	EllipticCurveSECP384R1 = 0x02
	// EllipticCurveSECP521R1 is the SECP521R1 elliptic curve.
	EllipticCurveSECP521R1 = 0x03
)

// ClientSecurity contains all the security settings of a client.
type ClientSecurity struct {
	Certificate       *x509.Certificate
	PrivateKey        *rsa.PrivateKey
	RemoteCertificate *x509.Certificate
	CipherSuites      []CipherSuite
	EllipticCurves    []EllipticCurve
}

// Validate the security.
func (s *ClientSecurity) Validate() error {
	if s.Certificate != nil {
		if s.PrivateKey == nil {
			return errors.New("a certificate was provided but not its associated private key")
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

func (s *ClientSecurity) supportedCipherSuites() []CipherSuite {
	if s.CipherSuites == nil {
		return []CipherSuite{
			CipherSuiteECDHERSAAES128GCMSHA256,
			CipherSuiteECDHERSAAES256GCMSHA384,
		}
	}

	return s.CipherSuites
}

func (s *ClientSecurity) supportedEllipticCurves() []EllipticCurve {
	if s.EllipticCurves == nil {
		return []EllipticCurve{
			EllipticCurveSECT571K1,
			EllipticCurveSECP384R1,
			EllipticCurveSECP521R1,
		}
	}

	return s.EllipticCurves
}
