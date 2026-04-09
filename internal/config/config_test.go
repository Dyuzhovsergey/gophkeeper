package config

import "testing"

func TestLoadServerDefaults(t *testing.T) {
	got, err := LoadServer(nil)
	if err != nil {
		t.Fatalf("LoadServer returned error: %v", err)
	}

	if got.RunAddress != defaultServerRunAddress {
		t.Fatalf("unexpected run address: got %q, want %q", got.RunAddress, defaultServerRunAddress)
	}

	if got.LogLevel != defaultLogLevel {
		t.Fatalf("unexpected log level: got %q, want %q", got.LogLevel, defaultLogLevel)
	}
}

func TestLoadServerEnvOverridesFlags(t *testing.T) {
	t.Setenv(envServerRunAddress, "127.0.0.1:9090")
	t.Setenv(envServerLogLevel, "WARN")

	got, err := LoadServer([]string{
		"-a", "localhost:8080",
		"-log-level", "debug",
	})
	if err != nil {
		t.Fatalf("LoadServer returned error: %v", err)
	}

	if got.RunAddress != "127.0.0.1:9090" {
		t.Fatalf("unexpected run address: got %q, want %q", got.RunAddress, "127.0.0.1:9090")
	}

	if got.LogLevel != "warn" {
		t.Fatalf("unexpected log level: got %q, want %q", got.LogLevel, "warn")
	}
}

func TestLoadClientDefaults(t *testing.T) {
	got, err := LoadClient(nil)
	if err != nil {
		t.Fatalf("LoadClient returned error: %v", err)
	}

	if got.ServerAddress != defaultClientServerAddress {
		t.Fatalf("unexpected server address: got %q, want %q", got.ServerAddress, defaultClientServerAddress)
	}

	if got.LogLevel != defaultLogLevel {
		t.Fatalf("unexpected log level: got %q, want %q", got.LogLevel, defaultLogLevel)
	}
}

func TestLoadClientEnvOverridesFlags(t *testing.T) {
	t.Setenv(envClientServerAddress, "https://example.com/api/")
	t.Setenv(envClientLogLevel, "ERROR")

	got, err := LoadClient([]string{
		"-server-address", "http://localhost:8080",
		"-log-level", "debug",
	})
	if err != nil {
		t.Fatalf("LoadClient returned error: %v", err)
	}

	if got.ServerAddress != "https://example.com/api" {
		t.Fatalf("unexpected server address: got %q, want %q", got.ServerAddress, "https://example.com/api")
	}

	if got.LogLevel != "error" {
		t.Fatalf("unexpected log level: got %q, want %q", got.LogLevel, "error")
	}
}

func TestLoadClientInvalidAddress(t *testing.T) {
	_, err := LoadClient([]string{
		"-server-address", "localhost:8080",
	})
	if err == nil {
		t.Fatal("expected error for invalid client server address")
	}
}
