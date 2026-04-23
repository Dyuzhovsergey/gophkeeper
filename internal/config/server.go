// Package config предоставляет конфигурацию клиента и сервера.
package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	defaultServerRunAddress      = "localhost:8080"
	defaultServerShutdownTimeout = 5 * time.Second

	envServerRunAddress      = "GOPHKEEPER_SERVER_RUN_ADDRESS"
	envServerLogLevel        = "GOPHKEEPER_SERVER_LOG_LEVEL"
	envServerDatabaseDSN     = "GOPHKEEPER_SERVER_DATABASE_DSN"
	envServerJWTSecret       = "GOPHKEEPER_SERVER_JWT_SECRET"
	envServerShutdownTimeout = "GOPHKEEPER_SERVER_SHUTDOWN_TIMEOUT"
)

// ServerConfig описывает конфигурацию серверного приложения.
type ServerConfig struct {
	// RunAddress — адрес, на котором сервер принимает HTTP-запросы.
	RunAddress string

	// LogLevel — уровень логирования.
	LogLevel string

	// DatabaseDSN — строка подключения к PostgreSQL.
	DatabaseDSN string

	// JWTSecret — секрет подписи JWT.
	JWTSecret string

	// ShutdownTimeout — максимальное время на корректное завершение HTTP-сервера.
	ShutdownTimeout time.Duration
}

// DefaultServerConfig возвращает серверную конфигурацию по умолчанию.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		RunAddress:      defaultServerRunAddress,
		LogLevel:        defaultLogLevel,
		DatabaseDSN:     "",
		JWTSecret:       "",
		ShutdownTimeout: defaultServerShutdownTimeout,
	}
}

// LoadServer загружает конфигурацию сервера из defaults, flags и env.
//
// Приоритет значений:
// 1. значения по умолчанию;
// 2. флаги командной строки;
// 3. переменные окружения.
func LoadServer(args []string) (ServerConfig, error) {
	cfg := DefaultServerConfig()

	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "HTTP server listen address")
	fs.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "logger level")
	fs.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "PostgreSQL DSN")
	fs.StringVar(&cfg.JWTSecret, "jwt-secret", cfg.JWTSecret, "JWT signing secret")
	fs.DurationVar(&cfg.ShutdownTimeout, "shutdown-timeout", cfg.ShutdownTimeout, "graceful shutdown timeout")

	if err := fs.Parse(args); err != nil {
		return ServerConfig{}, err
	}

	applyStringEnv(&cfg.RunAddress, envServerRunAddress)
	applyStringEnv(&cfg.LogLevel, envServerLogLevel)
	applyStringEnv(&cfg.DatabaseDSN, envServerDatabaseDSN)
	applyStringEnv(&cfg.JWTSecret, envServerJWTSecret)

	if raw, ok := os.LookupEnv(envServerShutdownTimeout); ok {
		raw = strings.TrimSpace(raw)
		if raw != "" {
			timeout, err := time.ParseDuration(raw)
			if err != nil {
				return ServerConfig{}, fmt.Errorf("parse server shutdown timeout: %w", err)
			}
			cfg.ShutdownTimeout = timeout
		}
	}

	cfg.RunAddress = strings.TrimSpace(cfg.RunAddress)
	cfg.LogLevel = normalizeLogLevel(cfg.LogLevel)
	cfg.DatabaseDSN = strings.TrimSpace(cfg.DatabaseDSN)
	cfg.JWTSecret = strings.TrimSpace(cfg.JWTSecret)

	if err := cfg.Validate(); err != nil {
		return ServerConfig{}, err
	}

	return cfg, nil
}

// Validate проверяет корректность серверной конфигурации.
func (c ServerConfig) Validate() error {
	if strings.TrimSpace(c.RunAddress) == "" {
		return fmt.Errorf("server run address is empty")
	}

	if strings.TrimSpace(c.LogLevel) == "" {
		return fmt.Errorf("server log level is empty")
	}

	if strings.TrimSpace(c.DatabaseDSN) == "" {
		return fmt.Errorf("server database dsn is empty")
	}

	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("server jwt secret is empty")
	}

	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("server shutdown timeout must be greater than zero")
	}

	return nil
}
