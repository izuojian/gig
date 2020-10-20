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
		_, _ = fmt.Fprintf(DefaultWriter, ConsoleFrontColorGreen+"[GIG-debug]"+ConsoleFrontColorWhite+format, values...)
	}
}

func DebugPrintInfo(format string, values ...interface{}) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		_, _ = fmt.Fprintf(DefaultWriter, ConsoleFrontColorGreen+"[GIG]"+format+ConsoleFrontColorWhite, values...)
	}
}

func DebugPrintError(format string, values ...interface{}) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		_, _ = fmt.Fprintf(DefaultWriter, ConsoleFrontColorRed+"[GIG]"+format+ConsoleFrontColorWhite, values...)
	}
}