package fscp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
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
	LocalSequenceNumber  SequenceNumber
	RemoteSequenceNumber SequenceNumber
	PublicKey            *ecdsa.PublicKey
	PrivateKey           []byte
	RemotePublicKey      *ecdsa.PublicKey
	LocalSessionKey      []byte
	RemoteSessionKey     []byte
	LocalIV              []byte
	RemoteIV             []byte
	LocalAEAD            cipher.AEAD
	RemoteAEAD           cipher.AEAD
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

	prf12(s.LocalSessionKey, k.Bytes(), []byte("session key"), s.LocalHostIdentifier[:])
	prf12(s.RemoteSessionKey, k.Bytes(), []byte("session key"), s.RemoteHostIdentifier[:])

	localBlock, err := aes.NewCipher(s.LocalSessionKey)

	if err != nil {
		return fmt.Errorf("failed to instanciate block cipher: %s", err)
	}

	s.LocalAEAD, err = cipher.NewGCM(localBlock)

	if err != nil {
		return fmt.Errorf("failed to instanciate GCM: %s", err)
	}

	remoteBlock, err := aes.NewCipher(s.RemoteSessionKey)

	if err != nil {
		return fmt.Errorf("failed to instanciate block cipher: %s", err)
	}

	s.RemoteAEAD, err = cipher.NewGCM(remoteBlock)

	if err != nil {
		return fmt.Errorf("failed to instanciate GCM: %s", err)
	}

	s.LocalIV = make([]byte, 8, 12)
	s.RemoteIV = make([]byte, 8, 12)

	prf12(s.LocalIV, k.Bytes(), []byte("nonce prefix"), s.LocalHostIdentifier[:])
	prf12(s.RemoteIV, k.Bytes(), []byte("nonce prefix"), s.RemoteHostIdentifier[:])

	// Preallocate the buffers so we can simply copy the sequence numbers
	// without any allocation later on.
	s.LocalIV = append(s.LocalIV, 0x00, 0x00, 0x00, 0x00)
	s.RemoteIV = append(s.RemoteIV, 0x00, 0x00, 0x00, 0x00)

	return nil
}

// Decrypt a ciphertext.
//
// This method is not thread-safe.
//
// ciphertext will be modified after the call, regardless of the outcome.
func (s *Session) Decrypt(msg *messageData) ([]byte, error) {
	if msg.SequenceNumber <= s.RemoteSequenceNumber {
		return nil, fmt.Errorf("outdated message: expected %d but got %d", s.RemoteSequenceNumber, msg.SequenceNumber)
	}

	// Sadly, the initial protocol design separates the GCM tag with the
	// ciphertext length... forcing us to recreate a buffer.
	msg.Ciphertext = append(msg.Ciphertext, msg.GCMTag[:]...)

	updateIV(s.RemoteIV, msg.SequenceNumber)

	data, err := s.RemoteAEAD.Open(msg.Ciphertext[:0], s.RemoteIV, msg.Ciphertext, nil)

	if err != nil {
		return nil, err
	}

	s.RemoteSequenceNumber = msg.SequenceNumber

	return data, nil
}

// Encrypt a cleartext.
//
// This method is not thread-safe.
func (s *Session) Encrypt(cleartext []byte) *messageData {
	s.LocalSequenceNumber++
	updateIV(s.LocalIV, s.LocalSequenceNumber)

	cleartext = s.LocalAEAD.Seal(cleartext[:0], s.LocalIV, cleartext, nil)

	return &messageData{
		SequenceNumber: s.LocalSequenceNumber,
		GCMTag:         cleartext[len(cleartext)-16:],
		Ciphertext:     cleartext[:len(cleartext)-16],
	}
}

func updateIV(iv []byte, sequenceNumber SequenceNumber) {
	binary.BigEndian.PutUint32(iv[8:], uint32(sequenceNumber))
}
