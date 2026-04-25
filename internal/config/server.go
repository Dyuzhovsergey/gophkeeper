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
	defaultServerRunAddress  = "localhost:8080"
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 30 * time.Second
	defaultWriteTimeout      = 30 * time.Second
	defaultIdleTimeout       = 60 * time.Second

	envServerRunAddress        = "GOPHKEEPER_SERVER_RUN_ADDRESS"
	envServerLogLevel          = "GOPHKEEPER_SERVER_LOG_LEVEL"
	envServerDatabaseDSN       = "GOPHKEEPER_SERVER_DATABASE_DSN"
	envServerJWTSecret         = "GOPHKEEPER_SERVER_JWT_SECRET"
	envServerReadHeaderTimeout = "GOPHKEEPER_SERVER_READ_HEADER_TIMEOUT"
	envServerReadTimeout       = "GOPHKEEPER_SERVER_READ_TIMEOUT"
	envServerWriteTimeout      = "GOPHKEEPER_SERVER_WRITE_TIMEOUT"
	envServerIdleTimeout       = "GOPHKEEPER_SERVER_IDLE_TIMEOUT"
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

	// ReadHeaderTimeout — максимальное время чтения HTTP-заголовков.
	ReadHeaderTimeout time.Duration

	// ReadTimeout — максимальное время чтения всего HTTP-запроса.
	ReadTimeout time.Duration

	// WriteTimeout — максимальное время записи HTTP-ответа.
	WriteTimeout time.Duration

	// IdleTimeout — максимальное время ожидания следующего запроса в keep-alive соединении.
	IdleTimeout time.Duration

	// ShutdownTimeout — максимальное время на корректное завершение HTTP-сервера.
	ShutdownTimeout time.Duration
}

// DefaultServerConfig возвращает серверную конфигурацию по умолчанию.

func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		RunAddress:        defaultServerRunAddress,
		LogLevel:          defaultLogLevel,
		DatabaseDSN:       "",
		JWTSecret:         "",
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
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
	fs.DurationVar(&cfg.ReadHeaderTimeout, "read-header-timeout", cfg.ReadHeaderTimeout, "HTTP read header timeout")
	fs.DurationVar(&cfg.ReadTimeout, "read-timeout", cfg.ReadTimeout, "HTTP read timeout")
	fs.DurationVar(&cfg.WriteTimeout, "write-timeout", cfg.WriteTimeout, "HTTP write timeout")
	fs.DurationVar(&cfg.IdleTimeout, "idle-timeout", cfg.IdleTimeout, "HTTP idle timeout")
	fs.DurationVar(&cfg.ShutdownTimeout, "shutdown-timeout", cfg.ShutdownTimeout, "graceful shutdown timeout")

	if err := fs.Parse(args); err != nil {
		return ServerConfig{}, err
	}

	applyStringEnv(&cfg.RunAddress, envServerRunAddress)
	applyStringEnv(&cfg.LogLevel, envServerLogLevel)
	applyStringEnv(&cfg.DatabaseDSN, envServerDatabaseDSN)
	applyStringEnv(&cfg.JWTSecret, envServerJWTSecret)

	if raw, ok := os.LookupEnv(envServerReadHeaderTimeout); ok {
		raw = strings.TrimSpace(raw)
		if raw != "" {
			value, err := time.ParseDuration(raw)
			if err != nil {
				return ServerConfig{}, fmt.Errorf("parse read header timeout: %w", err)
			}
			cfg.ReadHeaderTimeout = value
		}
	}

	if raw, ok := os.LookupEnv(envServerReadTimeout); ok {
		raw = strings.TrimSpace(raw)
		if raw != "" {
			value, err := time.ParseDuration(raw)
			if err != nil {
				return ServerConfig{}, fmt.Errorf("parse read timeout: %w", err)
			}
			cfg.ReadTimeout = value
		}
	}

	if raw, ok := os.LookupEnv(envServerWriteTimeout); ok {
		raw = strings.TrimSpace(raw)
		if raw != "" {
			value, err := time.ParseDuration(raw)
			if err != nil {
				return ServerConfig{}, fmt.Errorf("parse write timeout: %w", err)
			}
			cfg.WriteTimeout = value
		}
	}

	if raw, ok := os.LookupEnv(envServerIdleTimeout); ok {
		raw = strings.TrimSpace(raw)
		if raw != "" {
			value, err := time.ParseDuration(raw)
			if err != nil {
				return ServerConfig{}, fmt.Errorf("parse idle timeout: %w", err)
			}
			cfg.IdleTimeout = value
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

	if c.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("server read header timeout must be greater than zero")
	}

	if c.ReadTimeout <= 0 {
		return fmt.Errorf("server read timeout must be greater than zero")
	}

	if c.WriteTimeout <= 0 {
		return fmt.Errorf("server write timeout must be greater than zero")
	}

	if c.IdleTimeout <= 0 {
		return fmt.Errorf("server idle timeout must be greater than zero")
	}

	return nil
}
