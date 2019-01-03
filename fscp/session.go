package fscp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
)

// Session represents an outgoing or incoming session.
type Session struct {
	SessionNumber   SessionNumber
	CipherSuite     CipherSuite
	EllipticCurve   EllipticCurve
	SequenceNumber  SequenceNumber
	PublicKey       *ecdsa.PublicKey
	PrivateKey      []byte
	RemotePublicKey *ecdsa.PublicKey
	Key             []byte
}

// NewSession instantiate a new session.
//
// In case of an error, an invalid session is always returned.
func NewSession(sessionNumber SessionNumber, cipherSuite CipherSuite, ellipticCurve EllipticCurve) (*Session, error) {
	curve := ellipticCurve.Curve()

	// TODO: Instantiate a cipher and check, like we do already for the curve.

	if curve == nil {
		return &Session{
			SessionNumber: sessionNumber,
			CipherSuite:   NullCipherSuite,
			EllipticCurve: NullEllipticCurve,
		}, fmt.Errorf("unsupported elliptic curve: %s", ellipticCurve)
	}

	d, x, y, err := elliptic.GenerateKey(curve, rand.Reader)

	if err != nil {
		return nil, fmt.Errorf("failed to generate ECDHE key: %s", err)
	}

	publicKey := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}

	return &Session{
		SessionNumber: sessionNumber,
		CipherSuite:   cipherSuite,
		EllipticCurve: ellipticCurve,
		PublicKey:     publicKey,
		PrivateKey:    d,
	}, nil
}

// SetRemotePublicKey computes the session key.
func (s *Session) SetRemotePublicKey(publicKey *ecdsa.PublicKey) error {
	if s.RemotePublicKey != nil {
		if s.RemotePublicKey.Curve != publicKey.Curve ||
			s.RemotePublicKey.X.Cmp(publicKey.X) != 0 ||
			s.RemotePublicKey.Y.Cmp(publicKey.Y) != 0 {
			return errors.New("the remote public key was set previously to a different value")
		}

		return nil
	}

	curve := s.EllipticCurve.Curve()
	k, _ := curve.ScalarMult(publicKey.X, publicKey.Y, s.PrivateKey)

	s.RemotePublicKey = publicKey
	s.Key = k.Bytes()

	return nil
}
