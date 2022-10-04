package gig

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var writer io.Writer = os.Stdout

const (
	ConsoleFrontColorWhite  string = "\u001B[30m"
	ConsoleFrontColorRed    string = "\u001B[31m"
	ConsoleFrontColorGreen  string = "\u001B[32m"
	ConsoleFrontColorYellow string = "\u001B[33m"
	ConsoleFrontColorBlue   string = "\u001B[34m"
	ConsoleFrontColorPink   string = "\u001B[35m"
	ConsoleFrontColorCyan   string = "\u001B[36m"
	ConsoleFrontColorBlack  string = "\u001B[37m"
	ConsoleFrontColorReset  string = "\u001B[0m"
)

const (
	green   = "\033[97;42m"
	white   = "\033[90;47m"
	yellow  = "\033[90;43m"
	red     = "\033[97;41m"
	blue    = "\033[97;44m"
	magenta = "\033[97;45m"
	cyan    = "\033[97;46m"
	reset   = "\033[0m"
)

// 是否调试模式
func IsDebugging() bool {
	return gigMode == devModeCode
}

// debugPrint
func debugPrint(format string, values ...interface{}) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		_, _ = fmt.Fprintf(writer, ConsoleFrontColorGreen+"[GIG-debug]"+ConsoleFrontColorReset+format, values...)
	}
}

// ConsolePrint
func ConsolePrint(format string, values ...interface{}) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		_, _ = fmt.Fprintf(writer, ConsoleFrontColorGreen+"[GIG-debug]"+ConsoleFrontColorReset+format, values...)
	}
}

// ConsoleColorPrint
func ConsoleColorPrint(color, format string, values ...interface{}) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		_, _ = fmt.Fprintf(writer, color+"[GIG-debug]"+ConsoleFrontColorReset+format, values...)
	}
}

// ConsolePrintSuccess
func ConsolePrintSuccess(format string, values ...interface{}) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		_, _ = fmt.Fprintf(writer, ConsoleFrontColorGreen+"[GIG-debug]"+format+ConsoleFrontColorReset, values...)
	}
}

// ConsolePrintWarn
func ConsolePrintWarn(format string, values ...interface{}) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		_, _ = fmt.Fprintf(writer, ConsoleFrontColorYellow+"[GIG-debug]"+format+ConsoleFrontColorReset, values...)
	}
}

// ConsolePrintError
func ConsolePrintError(format string, values ...interface{}) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		_, _ = fmt.Fprintf(writer, ConsoleFrontColorRed+"[GIG-debug]"+format+ConsoleFrontColorReset, values...)
	}
}
