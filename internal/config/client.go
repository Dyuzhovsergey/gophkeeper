package config

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"
)

const (
	defaultClientServerAddress = "http://localhost:8080"

	envClientServerAddress = "GOPHKEEPER_CLIENT_SERVER_ADDRESS"
	envClientLogLevel      = "GOPHKEEPER_CLIENT_LOG_LEVEL"
)

// ClientConfig описывает конфигурацию клиентского приложения.
type ClientConfig struct {
	// ServerAddress — базовый адрес удалённого сервера.
	ServerAddress string

	// LogLevel — уровень логирования.
	LogLevel string
}

// DefaultClientConfig возвращает клиентскую конфигурацию по умолчанию.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		ServerAddress: defaultClientServerAddress,
		LogLevel:      defaultLogLevel,
	}
}

// LoadClient загружает конфигурацию клиента из defaults, flags и env.
//
// Приоритет значений:
// 1. значения по умолчанию;
// 2. флаги командной строки;
// 3. переменные окружения.
func LoadClient(args []string) (ClientConfig, error) {
	cfg := DefaultClientConfig()

	fs := flag.NewFlagSet("client", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.ServerAddress, "server-address", cfg.ServerAddress, "remote server base URL")
	fs.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "logger level")

	if err := fs.Parse(args); err != nil {
		return ClientConfig{}, err
	}

	applyStringEnv(&cfg.ServerAddress, envClientServerAddress)
	applyStringEnv(&cfg.LogLevel, envClientLogLevel)

	cfg.ServerAddress = strings.TrimRight(strings.TrimSpace(cfg.ServerAddress), "/")
	cfg.LogLevel = normalizeLogLevel(cfg.LogLevel)

	if err := cfg.Validate(); err != nil {
		return ClientConfig{}, err
	}

	return cfg, nil
}

// Validate проверяет корректность клиентской конфигурации.
func (c ClientConfig) Validate() error {
	if strings.TrimSpace(c.ServerAddress) == "" {
		return fmt.Errorf("client server address is empty")
	}

	parsedURL, err := url.Parse(c.ServerAddress)
	if err != nil {
		return fmt.Errorf("parse client server address: %w", err)
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("client server address must be absolute URL")
	}

	switch parsedURL.Scheme {
	case "http", "https":
	default:
		return fmt.Errorf("client server address must use http or https")
	}

	if strings.TrimSpace(c.LogLevel) == "" {
		return fmt.Errorf("client log level is empty")
	}

	return nil
}
