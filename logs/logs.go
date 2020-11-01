package logs

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"time"
)

type Logger = zap.SugaredLogger

// NewLogger
func NewLogger(w io.Writer) *zap.SugaredLogger {

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
		EncodeTime:       _timeEncoder,
		EncodeDuration:   zapcore.SecondsDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		EncodeName:       nil,
		ConsoleSeparator: "|",
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig), // 编码配置器配置, 输出Json格式：NewJSONEncoder
		zapcore.AddSync(w),                       // 日志输出位置
		zapcore.DebugLevel,                       // 日志级别
	)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(3)).Sugar()
}

// 自定义日志输出时间格式
func _timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}
