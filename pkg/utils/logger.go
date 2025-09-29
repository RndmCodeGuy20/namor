package utils

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const (
	grey  = "\033[90m"
	reset = "\033[0m"
)

// FieldEncoderConfig configures field printing behavior
type FieldEncoderConfig struct {
	// PrintFields enables printing fields on a new line
	PrintFields bool
	// FieldColor is the ANSI color code for fields (default: grey)
	FieldColor string
	// IndentFields controls JSON indentation for fields
	IndentFields bool
	// IndentPrefix is the prefix for each indentation level
	IndentPrefix string
	// IndentValue is the string used for each indentation level
	IndentValue string
}

// DefaultFieldEncoderConfig returns sensible defaults
func DefaultFieldEncoderConfig() FieldEncoderConfig {
	return FieldEncoderConfig{
		PrintFields:  true,
		FieldColor:   grey,
		IndentFields: true,
		IndentPrefix: "",
		IndentValue:  "  ",
	}
}

type fieldEncoder struct {
	zapcore.Encoder
	config FieldEncoderConfig
}

func (e *fieldEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf, err := e.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return buf, err
	}

	// Only process fields if printing is enabled
	if !e.config.PrintFields || len(fields) == 0 {
		return buf, nil
	}

	// Collect fields into a map
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(enc)
	}

	if len(enc.Fields) > 0 {
		var jsonBytes []byte
		var marshalErr error

		if e.config.IndentFields {
			jsonBytes, marshalErr = json.MarshalIndent(enc.Fields, e.config.IndentPrefix, e.config.IndentValue)
		} else {
			jsonBytes, marshalErr = json.Marshal(enc.Fields)
		}

		if marshalErr == nil {
			buf.AppendString("\n" + e.config.FieldColor + string(jsonBytes) + reset)
		}
	}

	return buf, nil
}

func (e *fieldEncoder) Clone() zapcore.Encoder {
	return &fieldEncoder{
		Encoder: e.Encoder.Clone(),
		config:  e.config,
	}
}

// fieldFilterCore wraps a core to intercept Write calls and separate fields
type fieldFilterCore struct {
	zapcore.Core
	encoder *fieldEncoder
	output  zapcore.WriteSyncer
	level   zap.AtomicLevel
}

func (c *fieldFilterCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Encode the entry WITHOUT fields for the main line
	buf, err := c.encoder.Encoder.EncodeEntry(entry, nil)
	if err != nil {
		return err
	}

	// Now add the fields separately if enabled
	if c.encoder.config.PrintFields && len(fields) > 0 {
		enc := zapcore.NewMapObjectEncoder()
		for _, f := range fields {
			f.AddTo(enc)
		}

		if len(enc.Fields) > 0 {
			var jsonBytes []byte
			var marshalErr error

			if c.encoder.config.IndentFields {
				jsonBytes, marshalErr = json.MarshalIndent(enc.Fields, c.encoder.config.IndentPrefix, c.encoder.config.IndentValue)
			} else {
				jsonBytes, marshalErr = json.Marshal(enc.Fields)
			}

			if marshalErr == nil {
				buf.AppendString(c.encoder.config.FieldColor + string(jsonBytes) + reset + "\n")
			}
		}
	}

	_, err = c.output.Write(buf.Bytes())
	buf.Free()
	return err
}

func (c *fieldFilterCore) With(fields []zapcore.Field) zapcore.Core {
	return &fieldFilterCore{
		Core:    c.Core.With(fields),
		encoder: c.encoder,
		output:  c.output,
		level:   c.level,
	}
}

func (c *fieldFilterCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

// LoggerConfig holds all configuration for the logger
type LoggerConfig struct {
	// Environment: "production", "development", or empty for auto-detect
	Environment string
	// LogLevel: DEBUG, INFO, WARN, ERROR, FATAL
	LogLevel zapcore.Level
	// EnableCaller adds caller information to logs
	EnableCaller bool
	// EnableStacktrace adds stacktraces for errors
	EnableStacktrace bool
	// StacktraceLevel is the minimum level for stacktraces
	StacktraceLevel zapcore.Level
	// TimeFormat for log timestamps
	TimeFormat string
	// FieldEncoder configuration
	FieldEncoder FieldEncoderConfig
}

// DefaultLoggerConfig returns default configuration
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Environment:      getEnvironment(),
		LogLevel:         getLogLevel(),
		EnableCaller:     false,
		EnableStacktrace: false,
		StacktraceLevel:  zapcore.ErrorLevel,
		TimeFormat:       "",
		FieldEncoder:     DefaultFieldEncoderConfig(),
	}
}

// Logger wraps zap.Logger with convenience methods
type Logger struct {
	*zap.Logger
	config LoggerConfig
}

// NewLogger creates a new logger instance with default config
func NewLogger() (*Logger, error) {
	return NewLoggerWithConfig(DefaultLoggerConfig())
}

// NewLoggerWithConfig creates a logger with custom configuration
func NewLoggerWithConfig(cfg LoggerConfig) (*Logger, error) {
	var zapConfig zap.Config

	if cfg.Environment == "production" || cfg.Environment == "prod" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeTime = getTimeEncoderFromConfig(cfg)
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		zapConfig.EncoderConfig.MessageKey = "msg"
		zapConfig.EncoderConfig.ConsoleSeparator = "    "
		zapConfig.EncoderConfig.EncodeName = func(name string, enc zapcore.PrimitiveArrayEncoder) {
			if name != "" {
				enc.AppendString("[" + name + "]")
			}
		}
	}

	zapConfig.Level = zap.NewAtomicLevelAt(cfg.LogLevel)

	// If field printing is enabled, we need to create a core that doesn't print fields inline
	var core zapcore.Core
	if cfg.FieldEncoder.PrintFields {
		// Create a custom core that filters out fields from inline printing
		baseEncoder := zapcore.NewConsoleEncoder(zapConfig.EncoderConfig)
		customEncoder := &fieldEncoder{
			Encoder: baseEncoder,
			config:  cfg.FieldEncoder,
		}
		// Use a core wrapper that passes empty fields to the base encoder
		core = &fieldFilterCore{
			Core:    zapcore.NewCore(customEncoder, zapcore.Lock(os.Stdout), zapConfig.Level),
			encoder: customEncoder,
			output:  zapcore.Lock(os.Stdout),
			level:   zapConfig.Level,
		}
	} else {
		// Normal core without field filtering
		baseEncoder := zapcore.NewConsoleEncoder(zapConfig.EncoderConfig)
		customEncoder := &fieldEncoder{
			Encoder: baseEncoder,
			config:  cfg.FieldEncoder,
		}
		core = zapcore.NewCore(customEncoder, zapcore.Lock(os.Stdout), zapConfig.Level)
	}

	// Build options
	var options []zap.Option
	if cfg.EnableCaller {
		options = append(options, zap.AddCaller())
	}
	if cfg.EnableStacktrace {
		options = append(options, zap.AddStacktrace(cfg.StacktraceLevel))
	}

	logger := zap.New(core, options...)

	return &Logger{
		Logger: logger,
		config: cfg,
	}, nil
}

// NewLoggerMust creates a new logger and panics on error
func NewLoggerMust() *Logger {
	logger, err := NewLogger()
	if err != nil {
		panic(err)
	}
	return logger
}

// NewLoggerWithFieldPrinting creates a logger with field printing enabled
func NewLoggerWithFieldPrinting(printFields bool) (*Logger, error) {
	cfg := DefaultLoggerConfig()
	cfg.FieldEncoder.PrintFields = printFields
	return NewLoggerWithConfig(cfg)
}

// getTimeEncoderFromConfig returns time encoder based on config
func getTimeEncoderFromConfig(cfg LoggerConfig) zapcore.TimeEncoder {
	if cfg.TimeFormat != "" {
		return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format(cfg.TimeFormat))
		}
	}

	// Check environment variable
	timeFormat := os.Getenv("LOG_TIME_FORMAT")
	if timeFormat != "" {
		return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format(timeFormat))
		}
	}

	if cfg.Environment == "production" || cfg.Environment == "prod" {
		return zapcore.ISO8601TimeEncoder
	}

	// Default development format
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}
}

// getLogLevel returns the log level from environment variable
func getLogLevel() zapcore.Level {
	level := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	switch level {
	case "DEBUG":
		return zapcore.DebugLevel
	case "INFO":
		return zapcore.InfoLevel
	case "WARN", "WARNING":
		return zapcore.WarnLevel
	case "ERROR":
		return zapcore.ErrorLevel
	case "FATAL":
		return zapcore.FatalLevel
	default:
		return zapcore.DebugLevel
	}
}

// getEnvironment returns the current environment
func getEnvironment() string {
	env := strings.ToLower(os.Getenv("ENV"))
	if env == "" {
		env = "development"
	}
	return env
}

// WithField adds a field to the logger context
func (l *Logger) WithField(key string, value any) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.Any(key, value)),
		config: l.config,
	}
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]any) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &Logger{
		Logger: l.Logger.With(zapFields...),
		config: l.config,
	}
}

// WithError adds an error field to the logger context
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.Error(err)),
		config: l.config,
	}
}

// Close syncs the logger (call this before application exit)
func (l *Logger) Close() {
	_ = l.Logger.Sync()
}

// GetConfig returns the logger configuration
func (l *Logger) GetConfig() LoggerConfig {
	return l.config
}

// ServiceLogger creates a service-specific logger with common fields
type ServiceLogger struct {
	*Logger
	serviceName string
}

// NewServiceLogger creates a logger for a specific service
func NewServiceLogger(serviceName string) *ServiceLogger {
	baseLogger := NewLoggerMust().WithField("service", serviceName)
	baseLogger.Logger = baseLogger.Logger.Named(serviceName)

	return &ServiceLogger{
		Logger:      baseLogger,
		serviceName: serviceName,
	}
}

// NewServiceLoggerWithConfig creates a service logger with custom config
func NewServiceLoggerWithConfig(serviceName string, cfg LoggerConfig) (*ServiceLogger, error) {
	baseLogger, err := NewLoggerWithConfig(cfg)
	if err != nil {
		return nil, err
	}

	baseLogger = baseLogger.WithField("service", serviceName)
	baseLogger.Logger = baseLogger.Logger.Named(serviceName)

	return &ServiceLogger{
		Logger:      baseLogger,
		serviceName: serviceName,
	}, nil
}

// WithOperation adds an operation context to the service logger
func (sl *ServiceLogger) WithOperation(operation string) *Logger {
	return sl.Logger.WithField("operation", operation)
}

// WithRequestID adds a request ID to the service logger
func (sl *ServiceLogger) WithRequestID(requestID string) *Logger {
	return sl.Logger.WithField("request_id", requestID)
}
