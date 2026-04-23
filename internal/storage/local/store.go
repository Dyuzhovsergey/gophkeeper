package local

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultAppDirName      = "gophkeeper"
	defaultSessionFile     = "session.json"
	defaultDirPermissions  = 0o700
	defaultFilePermissions = 0o600
)

// ErrSessionNotFound означает, что локальная сессия не найдена.
var ErrSessionNotFound = errors.New("local session not found")

// FileStore хранит клиентскую сессию в локальном JSON-файле.
type FileStore struct {
	path string
}

// NewFileStore создаёт файловое хранилище локальной сессии.
func NewFileStore(path string) (*FileStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("local session path is empty")
	}

	return &FileStore{path: path}, nil
}

// DefaultSessionPath возвращает путь по умолчанию для файла локальной сессии.
func DefaultSessionPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}

	return filepath.Join(configDir, defaultAppDirName, defaultSessionFile), nil
}

// Path возвращает путь к файлу локальной сессии.
func (s *FileStore) Path() string {
	return s.path
}

// SaveSession сохраняет клиентскую сессию на диск.
func (s *FileStore) SaveSession(session Session) error {
	if err := session.Validate(); err != nil {
		return fmt.Errorf("validate local session: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, defaultDirPermissions); err != nil {
		return fmt.Errorf("create local session directory: %w", err)
	}

	payload, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal local session: %w", err)
	}

	tempFile, err := os.CreateTemp(dir, "session-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp local session file: %w", err)
	}

	tempPath := tempFile.Name()

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(payload); err != nil {
		return fmt.Errorf("write temp local session file: %w", err)
	}

	if err := tempFile.Chmod(defaultFilePermissions); err != nil {
		return fmt.Errorf("chmod temp local session file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp local session file: %w", err)
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("rename temp local session file: %w", err)
	}

	return nil
}

// LoadSession загружает клиентскую сессию с диска.
func (s *FileStore) LoadSession() (*Session, error) {
	file, err := os.Open(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}

		return nil, fmt.Errorf("open local session file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var session Session
	if err := json.NewDecoder(file).Decode(&session); err != nil {
		return nil, fmt.Errorf("decode local session file: %w", err)
	}

	if err := session.Validate(); err != nil {
		return nil, fmt.Errorf("validate loaded local session: %w", err)
	}

	return &session, nil
}

// ClearSession удаляет локально сохранённую сессию.
func (s *FileStore) ClearSession() error {
	if err := os.Remove(s.path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("remove local session file: %w", err)
	}

	return nil
}
