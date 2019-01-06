package fscp

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
)

func mustReadCertificateFile(path string) (cert *x509.Certificate) {
	b, err := ioutil.ReadFile(path)

	if err != nil {
		panic(err)
	}

	block, _ := pem.Decode(b)
	cert, err = x509.ParseCertificate(block.Bytes)

	if err != nil {
		panic(err)
	}

	return
}

var CertificateAlice = mustReadCertificateFile("fixtures/alice.crt")
var SomeHostIdentifier = HostIdentifier{0x01, 0x02, 0x03, 0x04}

func makeECDSAPublicKey() *ecdsa.PublicKey {
	curve := elliptic.P384()
	_, x, y, err := elliptic.GenerateKey(curve, rand.Reader)

	if err != nil {
		panic(fmt.Errorf("failed to generate ECDHE key: %s", err))
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}
}

func makePEMPublicKey(key *ecdsa.PublicKey) []byte {
	derBytes, err := x509.MarshalPKIXPublicKey(key)

	if err != nil {
		panic(err)
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	}

	return pem.EncodeToMemory(block)
}

var SomePublicKey = makeECDSAPublicKey()
var SomePEMPublicKey = makePEMPublicKey(SomePublicKey)

func TestSerialization(t *testing.T) {
	type msgImpl interface {
		serializable
		fmt.Stringer
	}

	testCases := []struct {
		Message        msgImpl
		MessageType    MessageType
		Expected       []byte
		ExpectedString string
	}{
		{
			Message: &messageHello{
				UniqueNumber: 0x12345678,
			},
			MessageType: MessageTypeHelloRequest,
			Expected: []byte{
				0x03, 0x00, 0x00, 0x04,
				0x12, 0x34, 0x56, 0x78,
			},
			ExpectedString: "HELLO [unique_number:12345678]",
		},
		{
			Message: &messageHello{
				UniqueNumber: 0x12345678,
			},
			MessageType: MessageTypeHelloResponse,
			Expected: []byte{
				0x03, 0x01, 0x00, 0x04,
				0x12, 0x34, 0x56, 0x78,
			},
			ExpectedString: "HELLO [unique_number:12345678]",
		},
		{
			Message:     &messagePresentation{},
			MessageType: MessageTypePresentation,
			Expected: []byte{
				0x03, 0x02, 0x00, 0x02,
				0x00, 0x00,
			},
			ExpectedString: "PRESENTATION [cert:]",
		},
		{
			Message: &messagePresentation{
				Certificate: CertificateAlice,
			},
			MessageType: MessageTypePresentation,
			Expected: append(
				[]byte{
					0x03, 0x02, 0x05, 0xd6,
					0x05, 0xd4,
				},
				CertificateAlice.Raw...,
			),
			ExpectedString: "PRESENTATION [cert:CN=alice,O=Freelan,ST=Alsace,C=FR]",
		},
		{
			Message: &messageSessionRequest{
				SessionNumber:  0x22446688,
				HostIdentifier: SomeHostIdentifier,
				CipherSuites: []CipherSuite{
					ECDHERSAAES128GCMSHA256,
					ECDHERSAAES256GCMSHA384,
				},
				EllipticCurves: []EllipticCurve{
					SECT571K1,
					SECP384R1,
					SECP521R1,
				},
				Signature: []byte{0xaa, 0xbb},
			},
			MessageType: MessageTypeSessionRequest,
			Expected: []byte{
				0x03, 0x03, 0x00, 0x31,
				0x22, 0x44, 0x66, 0x88,
				0x01, 0x02, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x02, 0x01, 0x02, 0x00, 0x03, 0x01, 0x02, 0x03,
				0x00, 0x02, 0xaa, 0xbb,
			},
			ExpectedString: "SESSION_REQUEST [sid:22446688,hid:0102030400000000000000000000000000000000000000000000000000000000,ciphers:ECDHERSAAES128GCMSHA256,ECDHERSAAES256GCMSHA384,curves:SECT571K1,SECP384R1,SECP521R1]",
		},
		{
			Message: &messageSessionRequest{
				SessionNumber:  0x22446688,
				HostIdentifier: SomeHostIdentifier,
				CipherSuites: []CipherSuite{
					ECDHERSAAES128GCMSHA256,
					ECDHERSAAES256GCMSHA384,
				},
				EllipticCurves: []EllipticCurve{
					SECT571K1,
					SECP384R1,
					SECP521R1,
				},
				Signature: nil,
			},
			MessageType: MessageTypeSessionRequest,
			Expected: []byte{
				0x03, 0x03, 0x00, 0x2f,
				0x22, 0x44, 0x66, 0x88,
				0x01, 0x02, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x02, 0x01, 0x02, 0x00, 0x03, 0x01, 0x02, 0x03,
				0x00, 0x00,
			},
			ExpectedString: "SESSION_REQUEST [sid:22446688,hid:0102030400000000000000000000000000000000000000000000000000000000,ciphers:ECDHERSAAES128GCMSHA256,ECDHERSAAES256GCMSHA384,curves:SECT571K1,SECP384R1,SECP521R1]",
		},
		{
			Message: &messageSession{
				SessionNumber:  0x22446688,
				HostIdentifier: SomeHostIdentifier,
				CipherSuite:    ECDHERSAAES128GCMSHA256,
				EllipticCurve:  SECP384R1,
				PublicKey:      SomePublicKey,
				Signature:      []byte{0xaa, 0xbb},
			},
			MessageType: MessageTypeSession,
			Expected: append(
				append(
					[]byte{
						0x03, 0x04, 0x01, 0x05,
						0x22, 0x44, 0x66, 0x88,
						0x01, 0x02, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x01, 0x02, 0x00, 0x00,
						0x00, byte(len(SomePEMPublicKey)),
					},
					SomePEMPublicKey...,
				),
				[]byte{

					0x00, 0x02, 0xaa, 0xbb,
				}...,
			),
			ExpectedString: "SESSION [sid:22446688,hid:0102030400000000000000000000000000000000000000000000000000000000,cipher:ECDHERSAAES128GCMSHA256,curve:SECP384R1]",
		},
		{
			Message: &messageData{
				Channel:        0x02,
				SequenceNumber: 0x22446688,
				GCMTag: []byte{
					0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
					0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				},
				Ciphertext: []byte{0xaa, 0xbb},
			},
			MessageType: MessageTypeData,
			Expected: []byte{
				0x03, 0x72, 0x00, 0x18,
				0x22, 0x44, 0x66, 0x88,
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x00, 0x02, 0xaa, 0xbb,
			},
			ExpectedString: "DATA [ch:2,seq:22446688,clen:2]",
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%s", testCase.MessageType), func(t *testing.T) {
			buf := &bytes.Buffer{}

			if msg, _ := testCase.Message.(*messageData); msg != nil {
				writeDataMessage(buf, msg)
			} else {
				writeMessage(buf, testCase.MessageType, testCase.Message)
			}

			if bytes.Compare(buf.Bytes(), testCase.Expected) != 0 {
				t.Errorf("\n- %v\n+ %v", hex.EncodeToString(testCase.Expected), hex.EncodeToString(buf.Bytes()))
			}

			r := bytes.NewReader(buf.Bytes())
			mt, msg, err := readMessage(r)

			if err != nil {
				t.Fatalf("expected no error but got: %s", err)
			}

			if msg, _ := testCase.Message.(*messageData); msg != nil {
				emt := testCase.MessageType + MessageType(msg.Channel)

				if mt != emt {
					t.Errorf("expected: `%v`, got: `%v`", emt, mt)
				}
			} else {
				if mt != testCase.MessageType {
					t.Errorf("expected: `%v`, got: `%v`", testCase.MessageType, mt)
				}
			}

			if !reflect.DeepEqual(msg, testCase.Message) {
				t.Errorf("\n- %v\n+ %v", testCase.Message, msg)
			}

			t.Run("as string", func(t *testing.T) {
				str := testCase.Message.String()

				if str != testCase.ExpectedString {
					t.Errorf("\n- %v\n+ %v", testCase.ExpectedString, str)
				}
			})
		})
	}
}
