package fscp

import (
	"encoding/hex"
	"fmt"
	"os"
)

var debug = readDebug()

func readDebug() bool {
	return os.Getenv("FREELAN_FSCP_DEBUG") == "1"
}

func debugPrintf(msg string, args ...interface{}) {
	if debug {
		fmt.Fprintf(os.Stderr, msg, args...)
	}
}

type debugLenReader struct {
	lenReader
}

func (r debugLenReader) Read(b []byte) (int, error) {
	n, err := r.lenReader.Read(b)

	if err == nil {
		w := hex.Dumper(os.Stderr)
		w.Write(b)
		w.Close()
	}

	return n, err
}

func newDebugLenReader(r lenReader) debugLenReader {
	return debugLenReader{r}
}
