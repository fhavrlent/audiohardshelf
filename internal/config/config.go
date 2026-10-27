// Package config reads AudioHardShelf settings from the environment.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHardcoverAPIURL = "https://api.hardcover.app/v1/graphql"
	defaultSyncInterval    = time.Hour
	defaultAPITimeout      = 10 * time.Second
)

type Config struct {
	ABSURL          string
	ABSAPIKey       string
	HardcoverAPIURL string
	HardcoverAPIKey string
	// SyncInterval is how often to sync when no progress updates arrive.
	SyncInterval time.Duration
	APITimeout   time.Duration
	LogLevel     slog.Level
}

// Load reads the configuration from environment variables, preferring
// Docker secrets for API keys when they are mounted.
func Load() (Config, error) {
	syncInterval, err := parseSyncInterval(os.Getenv("SYNC_INTERVAL"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		ABSURL:          strings.TrimRight(os.Getenv("ABS_URL"), "/"),
		ABSAPIKey:       secretOrEnv("abs_api_key", "ABS_API_KEY"),
		HardcoverAPIURL: envOr("HARDCOVER_API_URL", defaultHardcoverAPIURL),
		HardcoverAPIKey: secretOrEnv("hardcover_api_key", "HARDCOVER_API_KEY"),
		SyncInterval:    syncInterval,
		APITimeout:      parseTimeout(os.Getenv("API_TIMEOUT_MS")),
		LogLevel:        parseLogLevel(os.Getenv("LOG_LEVEL")),
	}, nil
}

// Missing returns the names of required settings that are not set.
func (c Config) Missing() []string {
	var missing []string
	for _, setting := range []struct{ name, value string }{
		{"ABS_URL", c.ABSURL},
		{"ABS_API_KEY", c.ABSAPIKey},
		{"HARDCOVER_API_KEY", c.HardcoverAPIKey},
	} {
		if setting.value == "" {
			missing = append(missing, setting.name)
		}
	}
	return missing
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func secretOrEnv(secret, env string) string {
	if data, err := os.ReadFile("/run/secrets/" + secret); err == nil {
		if value := strings.TrimSpace(string(data)); value != "" {
			return value
		}
	}
	return os.Getenv(env)
}

func parseSyncInterval(value string) (time.Duration, error) {
	if value == "" {
		return defaultSyncInterval, nil
	}
	interval, err := time.ParseDuration(value)
	if err != nil || interval < time.Minute {
		return 0, fmt.Errorf("invalid SYNC_INTERVAL %q: use a duration of at least 1m, such as 30m or 1h", value)
	}
	return interval, nil
}

func parseTimeout(value string) time.Duration {
	ms, err := strconv.Atoi(value)
	if err != nil || ms <= 0 {
		return defaultAPITimeout
	}
	return time.Duration(ms) * time.Millisecond
}

func parseLogLevel(value string) slog.Level {
	level := slog.LevelDebug
	if value != "" {
		// Unknown values keep the default.
		_ = level.UnmarshalText([]byte(value))
	}
	return level
}
