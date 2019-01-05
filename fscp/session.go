package fscp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// Session represents an outgoing or incoming session.
type Session struct {
	LocalHostIdentifier  HostIdentifier
	RemoteHostIdentifier HostIdentifier
	SessionNumber        SessionNumber
	CipherSuite          CipherSuite
	EllipticCurve        EllipticCurve
	SequenceNumber       SequenceNumber
	PublicKey            *ecdsa.PublicKey
	PrivateKey           []byte
	RemotePublicKey      *ecdsa.PublicKey
	LocalSessionKey      []byte
	RemoteSessionKey     []byte
	LocalNOncePrefix     []byte
	RemoteNOncePrefix    []byte
}

// NewSession instantiate a new session.
//
// In case of an error, an invalid session is always returned.
func NewSession(hostIdentifier HostIdentifier, sessionNumber SessionNumber, cipherSuite CipherSuite, ellipticCurve EllipticCurve) (*Session, error) {
	curve := ellipticCurve.Curve()

	// TODO: Instantiate a cipher and check, like we do already for the curve.

	if curve == nil {
		return &Session{
			LocalHostIdentifier: hostIdentifier,
			SessionNumber:       sessionNumber,
			CipherSuite:         NullCipherSuite,
			EllipticCurve:       NullEllipticCurve,
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
		LocalHostIdentifier: hostIdentifier,
		SessionNumber:       sessionNumber,
		CipherSuite:         cipherSuite,
		EllipticCurve:       ellipticCurve,
		PublicKey:           publicKey,
		PrivateKey:          d,
	}, nil
}

// SetRemote computes the session keys.
func (s *Session) SetRemote(hostIdentifier HostIdentifier, publicKey *ecdsa.PublicKey) error {
	if s.RemotePublicKey != nil {
		if s.RemotePublicKey.Curve != publicKey.Curve ||
			s.RemotePublicKey.X.Cmp(publicKey.X) != 0 ||
			s.RemotePublicKey.Y.Cmp(publicKey.Y) != 0 {
			return errors.New("the remote public key was set previously to a different value")
		}

		return nil
	}

	s.RemoteHostIdentifier = hostIdentifier

	curve := s.EllipticCurve.Curve()

	// k should never be kept around for too long.
	//
	// We derive the keys from it and then discard it.
	k, _ := curve.ScalarMult(publicKey.X, publicKey.Y, s.PrivateKey)

	s.RemotePublicKey = publicKey

	s.LocalSessionKey = make([]byte, s.CipherSuite.BlockSize())
	s.RemoteSessionKey = make([]byte, s.CipherSuite.BlockSize())
	s.LocalNOncePrefix = make([]byte, 8)
	s.RemoteNOncePrefix = make([]byte, 8)

	prf12(s.LocalSessionKey, k.Bytes(), []byte("session key"), s.LocalHostIdentifier[:])
	prf12(s.RemoteSessionKey, k.Bytes(), []byte("session key"), s.RemoteHostIdentifier[:])
	prf12(s.LocalNOncePrefix, k.Bytes(), []byte("nonce prefix"), s.LocalHostIdentifier[:])
	prf12(s.RemoteNOncePrefix, k.Bytes(), []byte("nonce prefix"), s.RemoteHostIdentifier[:])

	fmt.Println("local session key: ", hex.EncodeToString(s.LocalSessionKey))
	fmt.Println("remote session key: ", hex.EncodeToString(s.RemoteSessionKey))

	return nil
}
