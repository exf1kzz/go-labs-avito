package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type HTTPConfig struct {
	Address           string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type DatabaseConfig struct {
	URL                   string
	MaxConnections        int32
	MinConnections        int32
	MaxConnectionLifetime time.Duration
	ConnectTimeout        time.Duration
	QueryTimeout          time.Duration
}

type Config struct {
	HTTP            HTTPConfig
	Database        DatabaseConfig
	LogLevel        string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	var config Config
	var err error

	config.HTTP.Address, err = requiredString("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}

	config.HTTP.ReadTimeout, err = requiredDuration("HTTP_READ_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	config.HTTP.ReadHeaderTimeout, err = requiredDuration("HTTP_READ_HEADER_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	config.HTTP.WriteTimeout, err = requiredDuration("HTTP_WRITE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	config.HTTP.IdleTimeout, err = requiredDuration("HTTP_IDLE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	config.LogLevel, err = requiredString("LOG_LEVEL")
	if err != nil {
		return Config{}, err
	}

	config.LogLevel = strings.ToLower(config.LogLevel)

	switch config.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf(
			"LOG_LEVEL must be one of debug, info, warn, error",
		)
	}

	config.ShutdownTimeout, err = requiredDuration("SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	config.Database.URL, err = requiredString("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	config.Database.MaxConnections, err = requiredInt32("DATABASE_MAX_CONNS")
	if err != nil {
		return Config{}, err
	}

	config.Database.MinConnections, err = requiredInt32("DATABASE_MIN_CONNS")
	if err != nil {
		return Config{}, err
	}

	config.Database.MaxConnectionLifetime, err = requiredDuration("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return Config{}, err
	}

	config.Database.ConnectTimeout, err = requiredDuration("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	config.Database.QueryTimeout, err = requiredDuration("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	if config.Database.MaxConnections <= 0 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be positive")
	}

	if config.Database.MinConnections > config.Database.MaxConnections {
		return Config{}, fmt.Errorf(
			"DATABASE_MIN_CONNS must not exceed DATABASE_MAX_CONNS",
		)
	}

	return config, nil
}

func requiredString(name string) (string, error) {
	value, ok := os.LookupEnv(name)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return value, nil
}

func requiredDuration(name string) (time.Duration, error) {
	value, err := requiredString(name)
	if err != nil {
		return 0, err
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid duration %q: %w", name, value, err)
	}

	if duration <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}

	return duration, nil
}

func requiredInt32(name string) (int32, error) {
	value, err := requiredString(name)
	if err != nil {
		return 0, err
	}

	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid integer %q: %w", name, value, err)
	}

	if number < 0 {
		return 0, fmt.Errorf("%s must not be negative", name)
	}

	return int32(number), nil
}
