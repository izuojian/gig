package logs

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
)

type Logger = zap.SugaredLogger

var (
	gigLogger *Logger
)

// 日志文件配置
type LogConfig struct {
	Dir        string
	File       string
	MaxSize    int
	MaxAge     int
	MaxBackups int
	Compress   bool
}

// SetGigLoggger
func SetGigLoggger(config LogConfig) {
	// 框架使用的日志
	gigLogger = NewLogger(config)
}

// NewLogger
func NewLogger(c LogConfig) *zap.SugaredLogger {
	if c.Dir != "" {
		if _, err := os.Stat(c.Dir); os.IsNotExist(err) {
			err = os.MkdirAll(c.Dir, os.ModePerm)
			if err != nil {
				panic("Create Log Dir failed")
			}
		}
	}

	// 日志分割设置
	hook := lumberjack.Logger{
		Filename:   filepath.Join(c.Dir, c.File),
		MaxSize:    c.MaxSize,
		MaxAge:     c.MaxAge,
		MaxBackups: c.MaxBackups,
		LocalTime:  true,
		Compress:   c.Compress,
	}

	// 日志格式配置
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:       "msg",
		LevelKey:         "level",
		TimeKey:          "time",
		NameKey:          "logger",
		CallerKey:        "caller",
		FunctionKey:      zapcore.OmitKey,
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      zapcore.CapitalLevelEncoder,
		EncodeTime:       zapcore.ISO8601TimeEncoder,
		EncodeDuration:   zapcore.SecondsDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		EncodeName:       nil,
		ConsoleSeparator: "|",
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig), // 编码配置器配置, 输出Json格式：NewConsoleEncoder
		zapcore.AddSync(&hook),                   // 打印到控制台和文件
		zap.NewAtomicLevelAt(zap.DebugLevel),     // 日志级别
	)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
}

// Debug
func Debug(template string, args ...interface{}) {
	gigLogger.Debugf(template, args...)
}

// Info
func Info(template string, args ...interface{}) {
	gigLogger.Infof(template, args...)
}

// Warn
func Warn(template string, args ...interface{}) {
	gigLogger.Warnf(template, args...)
}

// Error
func Error(template string, args ...interface{}) {
	gigLogger.Errorf(template, args...)
}

// DPanic development
func DPanic(template string, args ...interface{}) {
	gigLogger.DPanicf(template, args...)
}

// Panic
func Panic(template string, args ...interface{}) {
	gigLogger.Panicf(template, args...)
}

// Fatal
func Fatal(template string, args ...interface{}) {
	gigLogger.Fatalf(template, args...)
}
