package logger

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type contextKey string

const LoggerKey contextKey = "logger"

type Logger struct {
	*zerolog.Logger
}

// New creates a new logger instance with service context
func New(service string) *Logger {
	hostname, _ := os.Hostname()

	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = "@timestamp" // ELK compatible

	// Create logger with JSON output for production
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", service).
		Str("hostname", hostname).
		Str("environment", getEnv("ENVIRONMENT", "development")).
		Str("version", getEnv("SERVICE_VERSION", "unknown")).
		Logger()

	return &Logger{&logger}
}

// WithContext returns a logger from context or creates a new one
func WithContext(ctx context.Context, service string) *Logger {
	if logger, ok := ctx.Value(LoggerKey).(*Logger); ok {
		return logger
	}
	return New(service)
}

// ToContext adds logger to context
func (l *Logger) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, LoggerKey, l)
}

// WithRequestID adds request/correlation ID for tracing
func (l *Logger) WithRequestID(requestID string) *Logger {
	logger := l.Logger.With().Str("request_id", requestID).Logger()
	return &Logger{&logger}
}

// WithJob adds job context for cron jobs
func (l *Logger) WithJob(jobName string) *Logger {
	logger := l.Logger.With().
		Str("job_name", jobName).
		Str("job_type", "cron").
		Logger()
	return &Logger{&logger}
}

// WithEvent adds event context
func (l *Logger) WithEvent(eventID int, eventSlug string) *Logger {
	logger := l.Logger.With().
		Int("event_id", eventID).
		Str("event_slug", eventSlug).
		Logger()
	return &Logger{&logger}
}

// WithError adds error context
func (l *Logger) WithError(err error) *Logger {
	logger := l.Logger.With().Err(err).Logger()
	return &Logger{&logger}
}

// LogOddsChange logs odds movement with business metrics
func (l *Logger) LogOddsChange(eventID int, marketType string, outcome string, oldValue, newValue, changePercentage float64) {
	l.Info().
		Str("action", "odds_change").
		Int("event_id", eventID).
		Str("market_type", marketType).
		Str("outcome", outcome).
		Float64("old_value", oldValue).
		Float64("new_value", newValue).
		Float64("change_percentage", changePercentage).
		Bool("is_significant", changePercentage > 10).
		Msg("Odds movement detected")
}

// LogJobStart logs job execution start
func (l *Logger) LogJobStart(jobName string, schedule string) {
	l.Info().
		Str("action", "job_start").
		Str("job_name", jobName).
		Str("schedule", schedule).
		Msg("Starting job execution")
}

// LogJobComplete logs job completion with metrics
func (l *Logger) LogJobComplete(jobName string, duration time.Duration, itemsProcessed int, errors int) {
	l.Info().
		Str("action", "job_complete").
		Str("job_name", jobName).
		Dur("duration", duration).
		Int("items_processed", itemsProcessed).
		Int("error_count", errors).
		Bool("has_errors", errors > 0).
		Msg("Job execution completed")
}

// LogAPICall logs external API calls
func (l *Logger) LogAPICall(method, url string, statusCode int, duration time.Duration, err error) {
	event := l.Info()
	if err != nil {
		event = l.Error().Err(err)
	}

	event.
		Str("action", "api_call").
		Str("method", method).
		Str("url", url).
		Int("status_code", statusCode).
		Dur("duration", duration).
		Bool("success", err == nil).
		Msg("External API call")
}

// LogDatabaseOperation logs database operations
func (l *Logger) LogDatabaseOperation(operation string, table string, affectedRows int, duration time.Duration, err error) {
	event := l.Info()
	if err != nil {
		event = l.Error().Err(err)
	}

	event.
		Str("action", "db_operation").
		Str("operation", operation).
		Str("table", table).
		Int("affected_rows", affectedRows).
		Dur("duration", duration).
		Bool("success", err == nil).
		Msg("Database operation")
}

// Fatalf logs a fatal error and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Fatal().Msgf(format, args...)
}

// SetupLogger configures global log level based on environment
func SetupLogger() {
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Pretty logging for development
	if getEnv("ENVIRONMENT", "development") == "development" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.Kitchen}
		logger := zerolog.New(output).With().Timestamp().Logger()
		zerolog.DefaultContextLogger = &logger
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
