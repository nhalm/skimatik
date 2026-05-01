// Package config holds the flat Config struct plus group loaders
// (LoadLogging, LoadDatabase, LoadHTTP) that each command's run function
// composes. Validation, defaults, and cross-field constraints live here, not
// in handlers or services.
//
// Mirrors the canonical go-blueprint CONFIG.md shape so the example-app reads
// like a contributor-facing scaffold rather than a one-off demo.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/spf13/viper"
)

// Config is the flat configuration the example-app loads from environment
// variables. Fields are populated by the group loaders below.
type Config struct {
	DatabaseURL string
	HTTPPort    int
	LogLevel    string
	LogFormat   string
}

// LoadLogging reads LOG_LEVEL and LOG_FORMAT, validates them, and populates
// cfg. It also performs viper's one-time setup (.env + AutomaticEnv) so later
// loaders can read values without re-initializing.
func LoadLogging(cfg *Config) error {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	logLevel := viper.GetString("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	if !isValidLogLevel(logLevel) {
		return fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error (got %q)", logLevel)
	}

	logFormat := viper.GetString("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "text"
	}
	if !isValidLogFormat(logFormat) {
		return fmt.Errorf("LOG_FORMAT must be one of: text, json (got %q)", logFormat)
	}

	cfg.LogLevel = logLevel
	cfg.LogFormat = logFormat
	return nil
}

// LoadDatabase reads DATABASE_URL and populates cfg.
func LoadDatabase(cfg *Config) error {
	databaseURL := viper.GetString("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	cfg.DatabaseURL = databaseURL
	return nil
}

// LoadHTTP reads HTTP_PORT (or PORT for backward compatibility), validates
// the range, and populates cfg.
func LoadHTTP(cfg *Config) error {
	httpPort := viper.GetInt("HTTP_PORT")
	if httpPort == 0 {
		httpPort = viper.GetInt("PORT")
	}
	if httpPort == 0 {
		httpPort = 8080
	}
	if httpPort < 1 || httpPort > 65535 {
		return fmt.Errorf("HTTP_PORT must be 1-65535 (got %d)", httpPort)
	}
	cfg.HTTPPort = httpPort
	return nil
}

var validLogLevels = []string{"debug", "info", "warn", "error"}
var validLogFormats = []string{"text", "json"}

func isValidLogLevel(l string) bool  { return slices.Contains(validLogLevels, l) }
func isValidLogFormat(f string) bool { return slices.Contains(validLogFormats, f) }
