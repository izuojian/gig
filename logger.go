package gig

import (
	"github.com/izuojian/gig/logs"
	"net/http"
	"time"
)

// LogFormatter gives the signature of the formatter function passed to LoggerWithFormatter
type LogFormatter func(params LogFormatterParams) string

// LogFormatterParams is the structure any formatter will be handed when time to log comes
type LogFormatterParams struct {
	Request *http.Request

	// TimeStamp shows the time after the server returns a response.
	TimeStamp time.Time
	// StatusCode is HTTP response code.
	StatusCode int
	// Latency is how much time the server cost to process a certain request.
	Latency time.Duration
	// ClientIP equals Context's ClientIP method.
	ClientIP string
	// Method is the HTTP method given to the request.
	Method string
	// Path is a path the client requests.
	Path string
	// ErrorMessage is set if error has occurred in processing the request.
	ErrorMessage string
	// isTerm shows whether does gin's output descriptor refers to a terminal.
	isTerm bool
	// BodySize is the size of the Response Body
	BodySize int
	// Keys are the keys set on the request's context.
	Keys map[string]interface{}
}

// 控制台输出颜色控制
func (p *LogFormatterParams) StatusCodeColor() string {
	code := p.StatusCode

	switch {
	case code >= http.StatusOK && code < http.StatusMultipleChoices:
		return ConsoleFrontColorGreen
	case code >= http.StatusMultipleChoices && code < http.StatusBadRequest:
		return ConsoleFrontColorWhite
	case code >= http.StatusBadRequest && code < http.StatusInternalServerError:
		return ConsoleFrontColorYellow
	default:
		return ConsoleFrontColorRed
	}
}

// Logger中间件实例
func Logger() HandlerFunc {
	return func(c *Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		param := LogFormatterParams{
			Request: c.Request,
		}

		// Stop timer
		param.TimeStamp = time.Now()
		param.Latency = param.TimeStamp.Sub(start)

		param.ClientIP = c.ClientIP()
		param.Method = c.Request.Method
		param.StatusCode = c.StatusCode
		param.ErrorMessage = c.Errors.ByType(ErrorTypePrivate).String()

		// param.BodySize = c.Writer.Size()

		if raw != "" {
			path = path + "?" + raw
		}
		param.Path = path

		if IsDebugging() {
			ConsolePrint("%s%v |%3d| %13v | %15s |%-7s %#v%s%s",
				param.StatusCodeColor(),
				param.TimeStamp.Format("2006-01-02 15:04:05"),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Path,
				ConsoleFrontColorReset,
				param.ErrorMessage,
			)
		}

		logs.Info("%d|%v|%s|%s %s %s",
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	}
}
