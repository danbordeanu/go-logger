package logger

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
)

// CSugaredLogger is a superset of zap.SugaredLogger
type CSugaredLogger struct {
	zap.SugaredLogger
}

// CLogger is a superset of zap.Logger
type CLogger struct {
	zap.Logger
}

var logger *CLogger
var correlationIdContextKey string
var correlationIdFieldKey string

// WithContextCorrelationId returns an instance of the same logger with the correlation ID taken from the context added to it.
func (l *CLogger) WithContextCorrelationId(ctx context.Context) *CLogger {
	correlationId := ctx.Value(correlationIdContextKey)
	return l.WithCorrelationId(correlationId)
}

// WithContextCorrelationId returns an instance of the same logger with the correlation ID taken from the context added to it.
func (l *CSugaredLogger) WithContextCorrelationId(ctx context.Context) *CSugaredLogger {
	correlationId := ctx.Value(correlationIdContextKey)
	return l.WithCorrelationId(correlationId)
}

// WithCorrelationId returns an instance of the same logger with the correlation ID field added to it.
func (l *CLogger) WithCorrelationId(correlationId interface{}) *CLogger {
	if s, ok := correlationId.(string); ok {
		return &CLogger{*l.Logger.With(zap.Stringp(correlationIdFieldKey, &s))}
	}
	return l
}

// WithCorrelationId returns an instance of the same logger with the correlation ID field added to it.
func (l *CSugaredLogger) WithCorrelationId(correlationId interface{}) *CSugaredLogger {
	if s, ok := correlationId.(string); ok {
		return &CSugaredLogger{*l.SugaredLogger.With(zap.Stringp(correlationIdFieldKey, &s))}
	}
	return l
}

func (l *CLogger) With(args ...zap.Field) *CLogger {
	return &CLogger{*l.Logger.With(args...)}
}

func (l *CSugaredLogger) With(args ...interface{}) *CSugaredLogger {
	return &CSugaredLogger{*l.SugaredLogger.With(args...)}
}

// SugaredLogger returns an instance of the sugared logger. You must have initialized the logger prior to this call.
func SugaredLogger() *CSugaredLogger {
	if logger == nil {
		panic("logger not initialized. Call Init(ctx)")
	}
	l := logger.Sugar()
	return &CSugaredLogger{*l}
}

// Logger returns an instance of the sugar-free logger. You must have initialized the logger prior to this call.
func Logger() *CLogger {
	if logger == nil {
		panic("logger not initialized. Call Init(ctx)")
	}
	return logger
}

// SetCorrelationIdFieldKey sets the correlation ID field key in JSON responses. By default, it is "correlation_id"
func SetCorrelationIdFieldKey(key string) {
	if key == "" {
		return
	}
	correlationIdFieldKey = key
}

// SetCorrelationIdContextKey sets the correlation ID context key. By default, it is "correlation_id"
func SetCorrelationIdContextKey(key string) {
	if key == "" {
		return
	}
	correlationIdContextKey = key
}

// Init bootstraps the logger. You must call this method just once at the
// beginning of your application. The default log level is Info.
//
// Call Logger() or SugaredLogger() to retrieve an instance of the desired type
// of logger. Use WithCorrelationId() or WithContextCorrelationId() to add a
// correlation ID to your logs.
//
// If enableLogLevelEndpoint is true, then an HTTP endpoint on port 53835 at
// /logger is exposed which can be used to change the log level dynamically. See
// the Zap documentation for more information.
//
// If developmentMode is true, then the logLevel is set to Debug and caller
// fields are more explicit. Do not enable this in production.
func Init(ctx context.Context, enableLogLevelEndpoint, developmentMode bool) {
	if logger != nil {
		return
	}
	var (
		zapConfig     zap.Config
		encoderConfig zapcore.EncoderConfig
		atom          zap.AtomicLevel
		loggerMode    []string
	)

	correlationIdContextKey = "correlation_id"
	correlationIdFieldKey = "correlation_id"

	if developmentMode {
		loggerMode = append(loggerMode, "dev")
		atom = zap.NewAtomicLevelAt(zap.DebugLevel)
		encoderConfig = zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    "func",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeCaller:   zapcore.FullCallerEncoder,
		}
		zapConfig = zap.Config{
			Level:             atom,
			Development:       true,
			DisableCaller:     false,
			DisableStacktrace: false,
			Sampling:          nil,
			Encoding:          "json",
			EncoderConfig:     encoderConfig,
			OutputPaths:       []string{"stdout"},
			ErrorOutputPaths:  []string{"stdout"},
			InitialFields:     nil,
		}
	} else {
		loggerMode = append(loggerMode, "prod")
		atom = zap.NewAtomicLevelAt(zap.InfoLevel)
		encoderConfig = zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
		zapConfig = zap.Config{
			Level:             atom,
			Development:       false,
			DisableCaller:     false,
			DisableStacktrace: false,
			Sampling:          &zap.SamplingConfig{Initial: 100, Thereafter: 100},
			Encoding:          "json",
			EncoderConfig:     encoderConfig,
			OutputPaths:       []string{"stdout"},
			ErrorOutputPaths:  []string{"stdout"},
			InitialFields:     nil,
		}
	}

	if enableLogLevelEndpoint {
		loggerMode = append(loggerMode, "serveHttp")
		go func() {
			mux := http.NewServeMux()
			mux.Handle("/loglevel", atom)
			_ = http.ListenAndServe(":53835", mux)
		}()
	}

	l, err := zapConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("logger initalization error: %s", err.Error()))
	}

	l.Info("Logger initialized successfully", zap.Strings("logger_modes", loggerMode))
	if enableLogLevelEndpoint {
		l.Info("Logger HTTP Server active on :53835/loglevel")
	}

	logger = &CLogger{*l}
}

func (l *CSugaredLogger) Print(args ...interface{}) {
	l.Debug(args...)
}

func (l *CSugaredLogger) Println(args ...interface{}) {
	l.Debug(args...)
}

func (l *CSugaredLogger) Printf(format string, args ...interface{}) {
	l.Debugf(format, args...)
}

func (l *CSugaredLogger) Fatalln(args ...interface{}) {
	l.Fatal(args)
}
