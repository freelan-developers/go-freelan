package fscp

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"
)

func TestSerialization(t *testing.T) {
	testCases := []struct {
		Message       interface{}
		MessageType   MessageType
		Expected      []byte
		SerializeFunc func(*bytes.Buffer, interface{})
	}{
		{
			Message: &messageHello{
				UniqueNumber: 0x12345678,
			},
			MessageType: MessageTypeHelloRequest,
			Expected:    []byte{0x03, 0x00, 0x00, 0x04, 0x12, 0x34, 0x56, 0x78},
			SerializeFunc: func(buf *bytes.Buffer, msg interface{}) {
				writeHelloRequest(buf, msg.(*messageHello))
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%s", testCase.MessageType), func(t *testing.T) {
			buf := &bytes.Buffer{}
			testCase.SerializeFunc(buf, testCase.Message)

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
