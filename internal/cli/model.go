package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dyuzhovsergey/gophkeeper/internal/buildinfo"
	"github.com/Dyuzhovsergey/gophkeeper/internal/clientapi"
	"github.com/Dyuzhovsergey/gophkeeper/internal/storage/local"
)

type actionResultMsg struct {
	text string
}

type actionErrorMsg struct {
	err error
}

// Model описывает минимальную Bubble Tea model клиента.
type Model struct {
	api   *clientapi.Client
	store *local.FileStore
	args  []string

	output string
	err    error
}

// NewModel создаёт стартовую Bubble Tea model клиента.
func NewModel(api *clientapi.Client, store *local.FileStore, args []string) *Model {
	return &Model{
		api:   api,
		store: store,
		args:  args,
	}
}

// Init запускает выполнение выбранной команды.
func (m *Model) Init() tea.Cmd {
	return m.runAction()
}

// Update обрабатывает результат выполнения команды.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actionResultMsg:
		m.output = msg.text
		return m, tea.Quit

	case actionErrorMsg:
		m.err = msg.err
		return m, tea.Quit

	default:
		return m, nil
	}
}

// View возвращает вывод Bubble Tea model.
func (m *Model) View() string {
	if m.err != nil {
		return m.err.Error() + "\n"
	}

	if m.output == "" {
		return "working...\n"
	}

	return m.output + "\n"
}

// Err возвращает итоговую ошибку выполнения команды.
func (m *Model) Err() error {
	return m.err
}

func (m *Model) runAction() tea.Cmd {
	return func() tea.Msg {
		text, err := m.execute(context.Background())
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return actionResultMsg{text: text}
	}
}

func (m *Model) execute(ctx context.Context) (string, error) {
	if len(m.args) == 0 {
		return "", fmt.Errorf("client command is required")
	}

	switch m.args[0] {
	case "version":
		return buildinfo.Current().String(), nil

	case "register":
		if len(m.args) != 3 {
			return "", fmt.Errorf("usage: client register <login> <password>")
		}

		resp, err := m.api.Register(ctx, clientapi.RegisterRequest{
			Login:    m.args[1],
			Password: m.args[2],
		})
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("registered: login=%s id=%s", resp.Login, resp.ID), nil

	case "login":
		if len(m.args) != 3 {
			return "", fmt.Errorf("usage: client login <login> <password>")
		}

		resp, err := m.api.Login(ctx, clientapi.LoginRequest{
			Login:    m.args[1],
			Password: m.args[2],
		})
		if err != nil {
			return "", err
		}

		expiresAt, err := time.Parse(time.RFC3339, strings.TrimSpace(resp.ExpiresAt))
		if err != nil {
			return "", fmt.Errorf("parse login expires_at: %w", err)
		}

		meResp, err := m.api.Me(ctx, resp.Token)
		if err != nil {
			return "", err
		}

		session := local.Session{
			Token:     resp.Token,
			UserID:    meResp.UserID,
			SessionID: meResp.SessionID,
			ExpiresAt: expiresAt,
		}

		if err := m.store.SaveSession(session); err != nil {
			return "", fmt.Errorf("save local session: %w", err)
		}

		return fmt.Sprintf("login successful: user_id=%s session_id=%s", meResp.UserID, meResp.SessionID), nil

	case "logout":
		if len(m.args) != 1 {
			return "", fmt.Errorf("usage: client logout")
		}

		if err := m.store.ClearSession(); err != nil {
			return "", fmt.Errorf("clear local session: %w", err)
		}

		return "logout successful", nil

	default:
		return "", fmt.Errorf("unknown client command: %s", m.args[0])
	}
}

// IsNotLoggedInError показывает, что локальная сессия отсутствует.
func IsNotLoggedInError(err error) bool {
	return errors.Is(err, local.ErrSessionNotFound)
}
