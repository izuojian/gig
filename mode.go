package gig

const (
	DevMode = "dev"
	TestMode = "test"
	ProdMode = "prod"
)

const (
	devModeCode = iota
	testModeCode
	prodModeCode
)

// 当前版本
const VERSION = "0.0.3"

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

// Mode 返回当前运行模式
func Mode() string {
	return modeName
}