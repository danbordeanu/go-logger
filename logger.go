package logger

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
)

type CSugaredLogger struct {
	zap.SugaredLogger
}

type CLogger struct {
	zap.Logger
}

var logger *CLogger

func (l *CLogger) WithCorrelationId(correlationId interface{}) *CLogger {
	if s, ok := correlationId.(string); ok {
		return &CLogger{*l.With(zap.Stringp("correlation_id", &s))}
	}
	return l
}

func (l *CSugaredLogger) WithCorrelationId(correlationId interface{}) *CSugaredLogger {
	if s, ok := correlationId.(string); ok {
		return &CSugaredLogger{*l.With(zap.Stringp("correlation_id", &s))}
	}
	return l
}

func SugaredLogger() *CSugaredLogger {
	if logger == nil {
		panic("logger not initialized. Call Init(ctx)")
	}
	l := logger.Sugar()
	return &CSugaredLogger{*l}
}

func Logger() *CLogger {
	if logger == nil {
		panic("logger not initialized. Call Init(ctx)")
	}
	return logger
}

func Init(ctx context.Context) {
	if logger != nil {return}
	var (
		zapConfig zap.Config
		encoderConfig zapcore.EncoderConfig
		atom zap.AtomicLevel
		loggerMode []string
		lsh bool
	)

	if devEnv, ok := ctx.Value("env_development").(bool); ok && devEnv {
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

	if lsh, ok := ctx.Value("logger_serveHttp").(bool); ok && lsh {
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
	if lsh {
		l.Info("Logger HTTP Server active on :53835/loglevel")
	}

	logger = &CLogger{*l}
}
