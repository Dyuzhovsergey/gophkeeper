package logger

import (
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    zapcore.Level
		wantErr bool
	}{
		{
			name:    "debug",
			input:   "debug",
			want:    zapcore.DebugLevel,
			wantErr: false,
		},
		{
			name:    "info",
			input:   "info",
			want:    zapcore.InfoLevel,
			wantErr: false,
		},
		{
			name:    "warn",
			input:   "warn",
			want:    zapcore.WarnLevel,
			wantErr: false,
		},
		{
			name:    "warning",
			input:   "warning",
			want:    zapcore.WarnLevel,
			wantErr: false,
		},
		{
			name:    "error",
			input:   "error",
			want:    zapcore.ErrorLevel,
			wantErr: false,
		},
		{
			name:    "with spaces and upper case",
			input:   " ERROR ",
			want:    zapcore.ErrorLevel,
			wantErr: false,
		},
		{
			name:    "invalid",
			input:   "trace",
			want:    zapcore.InfoLevel,
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			want:    zapcore.InfoLevel,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLevel(tt.input)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected level: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInit(t *testing.T) {
	log, err := Init("info")
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
}
