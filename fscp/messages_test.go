package fscp

import (
	"bytes"
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

func TestSerialization(t *testing.T) {
	testCases := []struct {
		Message     serializable
		MessageType MessageType
		Expected    []byte
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
		},
		{
			Message:     &messagePresentation{},
			MessageType: MessageTypePresentation,
			Expected: []byte{
				0x03, 0x02, 0x00, 0x02,
				0x00, 0x00,
			},
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
		},
		{
			Message: &messageSessionRequest{
				SessionNumber:  0x22446688,
				HostIdentifier: 0x12345678,
				CipherSuites: []CipherSuite{
					CipherSuiteECDHERSAAES128GCMSHA256,
					CipherSuiteECDHERSAAES256GCMSHA384,
				},
				EllipticCurves: []EllipticCurve{
					EllipticCurveSECT571K1,
					EllipticCurveSECP384R1,
					EllipticCurveSECP521R1,
				},
				Signature: []byte{0xaa, 0xbb},
			},
			MessageType: MessageTypeSessionRequest,
			Expected: []byte{
				0x03, 0x03, 0x00, 0x15,
				0x22, 0x44, 0x66, 0x88, 0x12, 0x34, 0x56, 0x78,
				0x00, 0x02, 0x01, 0x02, 0x00, 0x03, 0x01, 0x02, 0x03,
				0x00, 0x02, 0xaa, 0xbb,
			},
		},
		{
			Message: &messageSession{
				SessionNumber:  0x22446688,
				HostIdentifier: 0x12345678,
				CipherSuite:    CipherSuiteECDHERSAAES128GCMSHA256,
				EllipticCurve:  EllipticCurveSECT571K1,
				PublicKey:      []byte{0xab, 0xcd},
				Signature:      []byte{0xaa, 0xbb},
			},
			MessageType: MessageTypeSession,
			Expected: []byte{
				0x03, 0x04, 0x00, 0x14,
				0x22, 0x44, 0x66, 0x88, 0x12, 0x34, 0x56, 0x78,
				0x01, 0x01, 0x00, 0x00,
				0x00, 0x02, 0xab, 0xcd,
				0x00, 0x02, 0xaa, 0xbb,
			},
		},
		{
			Message: &messageData{
				Channel:        0x02,
				SequenceNumber: 0x22446688,
				GCMTag: [16]byte{
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
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%s", testCase.MessageType), func(t *testing.T) {
			buf := &bytes.Buffer{}

			if msg, _ := testCase.Message.(*messageData); msg != nil {
				writeDataMessage(buf, msg.Channel, msg)
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
		})
	}
}
