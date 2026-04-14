// Package config предоставляет конфигурацию клиента и сервера.
package config

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

const (
	defaultServerRunAddress = "localhost:8080"

	envServerRunAddress  = "GOPHKEEPER_SERVER_RUN_ADDRESS"
	envServerLogLevel    = "GOPHKEEPER_SERVER_LOG_LEVEL"
	envServerDatabaseDSN = "GOPHKEEPER_SERVER_DATABASE_DSN"
)

// ServerConfig описывает конфигурацию серверного приложения.
type ServerConfig struct {
	// RunAddress — адрес, на котором сервер принимает HTTP-запросы.
	RunAddress string

	// LogLevel — уровень логирования.
	LogLevel string

	// DatabaseDSN — строка подключения к PostgreSQL.
	DatabaseDSN string
}

// DefaultServerConfig возвращает серверную конфигурацию по умолчанию.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		RunAddress:  defaultServerRunAddress,
		LogLevel:    defaultLogLevel,
		DatabaseDSN: "",
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

	if err := fs.Parse(args); err != nil {
		return ServerConfig{}, err
	}

	applyStringEnv(&cfg.RunAddress, envServerRunAddress)
	applyStringEnv(&cfg.LogLevel, envServerLogLevel)
	applyStringEnv(&cfg.DatabaseDSN, envServerDatabaseDSN)

	cfg.RunAddress = strings.TrimSpace(cfg.RunAddress)
	cfg.LogLevel = normalizeLogLevel(cfg.LogLevel)
	cfg.DatabaseDSN = strings.TrimSpace(cfg.DatabaseDSN)

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

	return nil
}
