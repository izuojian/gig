package gig

import (
	"fmt"
	"strings"
)

func IsDebugging() bool {
	return gigMode == devModeCode
}

func debugPrint(format string, values ...interface{}) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		fmt.Fprintf(DefaultWriter, "[GIG-debug]" + format, values...)
	}
}