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
			Expected:    []byte{0x03, 0x00, 0x00, 0x04, 0x12, 0x34, 0x56, 0x78},
		},
		{
			Message: &messageHello{
				UniqueNumber: 0x12345678,
			},
			MessageType: MessageTypeHelloResponse,
			Expected:    []byte{0x03, 0x01, 0x00, 0x04, 0x12, 0x34, 0x56, 0x78},
		},
		{
			Message:     &messagePresentation{},
			MessageType: MessageTypePresentation,
			Expected:    []byte{0x03, 0x02, 0x00, 0x02, 0x00, 0x00},
		},
		{
			Message: &messagePresentation{
				Certificate: CertificateAlice,
			},
			MessageType: MessageTypePresentation,
			Expected:    append([]byte{0x03, 0x02, 0x05, 0xd6, 0x05, 0xd4}, CertificateAlice.Raw...),
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%s", testCase.MessageType), func(t *testing.T) {
			buf := &bytes.Buffer{}
			writeMessage(buf, testCase.MessageType, testCase.Message)

			if bytes.Compare(buf.Bytes(), testCase.Expected) != 0 {
				t.Errorf("\n- %v\n+ %v", hex.EncodeToString(testCase.Expected), hex.EncodeToString(buf.Bytes()))
			}

			r := bytes.NewReader(buf.Bytes())
			mt, msg, err := readMessage(r)

			if err != nil {
				t.Fatalf("expected no error but got: %s", err)
			}

			if mt != testCase.MessageType {
				t.Errorf("expected: `%v`, got: `%v`", testCase.MessageType, mt)
			}

			if !reflect.DeepEqual(msg, testCase.Message) {
				t.Errorf("\n- %v\n+ %v", testCase.Message, msg)
			}
		})
	}
}
