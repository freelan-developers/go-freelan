package fscp

import (
	"fmt"
	"os"
)

var debug = readDebug()

func readDebug() bool {
	return os.Getenv("FREELAN_FSCP_DEBUG") == "1"
}

func debugPrint(msg string, args ...interface{}) {
	if debug {
		fmt.Fprintf(os.Stderr, msg, args...)
	}
}
