package gig

import (
	"io"
	"os"
)

const EnvGigMode = "GIG_MODE"

const (
	DevMode  = "dev"
	TestMode = "test"
	ProdMode = "prod"
)

const (
	devModeCode = iota
	testModeCode
	prodModeCode
)

// VERSION represent beego web framework version.
const VERSION = "0.0.1"

// DefaultWriter is the default io.Writer used by Gin for debug output and
// middleware output like Logger() or Recovery().
// Note that both Logger and Recovery provides custom ways to configure their
// output io.Writer.
// To support coloring in Windows use:
// 		import "github.com/mattn/go-colorable"
// 		gin.DefaultWriter = colorable.NewColorableStdout()
var DefaultWriter io.Writer = os.Stdout

// DefaultErrorWriter is the default io.Writer used by Gin to debug errors
var DefaultErrorWriter io.Writer = os.Stderr

var gigMode = devModeCode
var modeName = DevMode

// 设置运行模式
func SetMode(value string) {
	if value == "" {
		value = DevMode
	}
	switch value {
	case DevMode:
		gigMode = devModeCode
	case TestMode:
		gigMode = testModeCode
	case ProdMode:
		gigMode = prodModeCode
	default:
		panic("gig mode unknown: " + value)
	}
	modeName = value
}

// Mode returns currently gin mode.
func Mode() string {
	return modeName
}
