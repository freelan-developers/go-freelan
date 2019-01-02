package fscp

import (
	"bytes"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
)

// Session represents an outgoing or incoming session.
type Session struct {
	SessionNumber   SessionNumber
	CipherSuite     CipherSuite
	EllipticCurve   EllipticCurve
	SequenceNumber  SequenceNumber
	PublicKey       []byte
	PrivateKey      []byte
	RemotePublicKey []byte
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

	block := &pem.Block{
		Type: "PUBLIC KEY",
		//FIXME: This should be ASN1 encoded.
		Bytes: elliptic.Marshal(curve, x, y),
	}
	publicKey := pem.EncodeToMemory(block)

	return &Session{
		SessionNumber: sessionNumber,
		CipherSuite:   cipherSuite,
		EllipticCurve: ellipticCurve,
		PublicKey:     publicKey,
		PrivateKey:    d,
	}, nil
}

// SetRemotePublicKey computes the session key.
func (s *Session) SetRemotePublicKey(publicKey []byte) error {
	if s.RemotePublicKey != nil {
		if !bytes.Equal(s.RemotePublicKey, publicKey) {
			return errors.New("the remote public key was set previously to a different value")
		}

		return nil
	}

	curve := s.EllipticCurve.Curve()
	x, y := elliptic.Unmarshal(curve, publicKey)
	k, _ := curve.ScalarMult(x, y, s.PrivateKey)

	s.RemotePublicKey = publicKey
	s.Key = k.Bytes()

	return nil
}
