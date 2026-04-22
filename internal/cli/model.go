package cli

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dyuzhovsergey/gophkeeper/internal/buildinfo"
	"github.com/Dyuzhovsergey/gophkeeper/internal/clientapi"
	"github.com/Dyuzhovsergey/gophkeeper/internal/storage/local"
)

// actionResultMsg передаёт в модель успешный результат выполнения действия.
type actionResultMsg struct {
	text string
}

// actionErrorMsg передаёт в модель ошибку выполнения действия.
type actionErrorMsg struct {
	err error
}

// sessionStatusMsg передаёт в модель информацию о состоянии локальной сессии.
type sessionStatusMsg struct {
	hasLocalSession bool
	sessionValid    bool
	userID          string
	sessionID       string
	expiresAt       time.Time
	status          string
	showMessage     string
}

// secretsListMsg передаёт в модель загруженный список секретов пользователя.
type secretsListMsg struct {
	items []clientapi.SecretResponse
}

// secretDetailsMsg передаёт в модель детали одного секрета.
type secretDetailsMsg struct {
	item clientapi.SecretResponse
}

// uiScreen описывает текущий экран TUI-клиента.
type uiScreen int

const (
	screenMenu uiScreen = iota
	screenRegister
	screenLogin
	screenMessage
	screenCreateCardSecret
	screenCreateBinarySecret
	screenSecretsList
	screenSecretDetails
	screenDeleteConfirm
	screenCreateTextSecret
	screenCreateCredentialsSecret
	screenUpdateTextSecret
	screenUpdateCredentialsSecret
	screenUpdateCardSecret
)

var menuItems = []string{
	"register",
	"login",
	"logout",
	"me",
	"secrets",
	"add text",
	"add credentials",
	"add card",
	"add file",
	"version",
	"quit",
}

// Model описывает Bubble Tea model клиента.
type Model struct {
	api   *clientapi.Client
	store *local.FileStore
	args  []string

	output string
	err    error

	screen    uiScreen
	menuIndex int

	inputs     []textinput.Model
	focusIndex int
	busy       bool
	message    string

	hasLocalSession  bool
	sessionValid     bool
	sessionUserID    string
	sessionSessionID string
	sessionExpiresAt time.Time
	sessionStatus    string

	// secrets хранит последний загруженный список секретов.
	secrets []clientapi.SecretResponse

	// secretsIndex хранит индекс выбранного секрета в списке.
	secretsIndex int

	// secretDetails хранит текущий открытый секрет.
	secretDetails *clientapi.SecretResponse

	// editingSecretID хранит идентификатор секрета, который сейчас редактируется.
	editingSecretID string
}

// NewModel создаёт стартовую Bubble Tea model клиента.
func NewModel(api *clientapi.Client, store *local.FileStore, args []string) *Model {
	return &Model{
		api:       api,
		store:     store,
		args:      args,
		screen:    screenMenu,
		menuIndex: 0,
	}
}

// Init запускает выполнение выбранной команды.
func (m *Model) Init() tea.Cmd {
	if m.isCommandMode() {
		return m.runAction()
	}

	return tea.Batch(
		textinput.Blink,
		m.loadSessionStatusCmd(),
	)
}

// Update обрабатывает события Bubble Tea.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.isCommandMode() {
		return m.updateCommandMode(msg)
	}

	return m.updateTUIMode(msg)
}

// View возвращает вывод Bubble Tea model.
func (m *Model) View() string {
	if m.isCommandMode() {
		if m.err != nil {
			return m.err.Error() + "\n"
		}

		if m.output == "" {
			return "working...\n"
		}

		return m.output + "\n"
	}

	switch m.screen {
	case screenMenu:
		return m.viewMenu()

	case screenRegister:
		return m.viewForm("Register")

	case screenLogin:
		return m.viewForm("Login")

	case screenMessage:
		if m.busy {
			return "working...\n"
		}

		return m.message + "\n\nPress Enter or Esc to return to menu.\n"

	case screenSecretsList:
		if m.busy {
			return "loading secrets...\n"
		}

		return m.viewSecretsList()

	case screenCreateTextSecret:
		return m.viewSecretForm("Create text secret")

	case screenCreateCredentialsSecret:
		return m.viewSecretForm("Create credentials secret")

	case screenCreateCardSecret:
		return m.viewSecretForm("Create card secret")

	case screenCreateBinarySecret:
		return m.viewSecretForm("Create binary secret")

	case screenUpdateTextSecret:
		return m.viewSecretForm("Update text secret")

	case screenUpdateCredentialsSecret:
		return m.viewSecretForm("Update credentials secret")

	case screenUpdateCardSecret:
		return m.viewSecretForm("Update card secret")

	case screenSecretDetails:
		if m.busy {
			return "loading secret details...\n"
		}

		return m.viewSecretDetails()

	default:
		return ""
	}
}

// Err возвращает итоговую ошибку выполнения команды.
func (m *Model) Err() error {
	return m.err
}

// isCommandMode показывает, запущен ли клиент в режиме команды с аргументами.
func (m *Model) isCommandMode() bool {
	return len(m.args) > 0
}

// updateCommandMode обрабатывает сообщения Bubble Tea в режиме запуска с аргументами.
func (m *Model) updateCommandMode(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// updateTUIMode обрабатывает сообщения Bubble Tea в интерактивном TUI-режиме.
func (m *Model) updateTUIMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actionResultMsg:
		m.busy = false
		m.screen = screenMessage
		m.message = msg.text
		return m, nil

	case actionErrorMsg:
		m.busy = false
		m.screen = screenMessage
		m.message = "Error: " + msg.err.Error()
		return m, nil
	case sessionStatusMsg:
		m.busy = false
		m.hasLocalSession = msg.hasLocalSession
		m.sessionValid = msg.sessionValid
		m.sessionUserID = msg.userID
		m.sessionSessionID = msg.sessionID
		m.sessionExpiresAt = msg.expiresAt
		m.sessionStatus = msg.status

		if msg.showMessage != "" {
			m.screen = screenMessage
			m.message = msg.showMessage
		}
		return m, nil

	case secretsListMsg:
		m.busy = false
		m.screen = screenSecretsList
		m.secrets = msg.items
		m.secretsIndex = 0
		return m, nil

	case secretDetailsMsg:
		m.busy = false
		m.screen = screenSecretDetails
		m.secretDetails = &msg.item
		return m, nil

	case tea.KeyMsg:
		switch m.screen {
		case screenMenu:
			return m.updateMenu(msg)

		case screenRegister,
			screenLogin,
			screenCreateTextSecret,
			screenCreateCredentialsSecret,
			screenCreateCardSecret,
			screenUpdateTextSecret,
			screenUpdateCredentialsSecret,
			screenUpdateCardSecret:
			model, cmd, handled := m.updateForm(msg)
			if handled {
				return model, cmd
			}

		case screenSecretsList:
			return m.updateSecretsList(msg)

		case screenSecretDetails:
			return m.updateSecretDetails(msg)

		case screenMessage:
			switch msg.String() {
			case "enter", "esc":
				m.screen = screenMenu
				m.message = ""
				return m, nil
			case "ctrl+c", "q":
				return m, tea.Quit
			}
		}
	}

	if m.screen == screenRegister ||
		m.screen == screenLogin ||
		m.screen == screenCreateTextSecret ||
		m.screen == screenCreateCredentialsSecret ||
		m.screen == screenCreateCardSecret ||
		m.screen == screenUpdateTextSecret ||
		m.screen == screenUpdateCredentialsSecret ||
		m.screen == screenUpdateCardSecret {
		var cmds []tea.Cmd

		for i := range m.inputs {
			var cmd tea.Cmd
			m.inputs[i], cmd = m.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)
	}

	return m, nil
}

// updateSecretsList обрабатывает клавиши на экране списка секретов.
func (m *Model) updateSecretsList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc":
		m.screen = screenMenu
		return m, nil

	case "up", "k":
		if m.secretsIndex > 0 {
			m.secretsIndex--
		}
		return m, nil

	case "down", "j":
		if m.secretsIndex < len(m.secrets)-1 {
			m.secretsIndex++
		}
		return m, nil

	case "r":
		m.busy = true
		return m, m.runListSecretsCmd()

	case "enter":
		if len(m.secrets) == 0 {
			return m, nil
		}

		m.busy = true
		return m, m.runGetSecretDetailsCmd(m.secrets[m.secretsIndex].ID)
	}

	return m, nil
}

// updateSecretDetails обрабатывает клавиши на экране деталей секрета.
func (m *Model) updateSecretDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc", "enter":
		m.screen = screenSecretsList
		return m, nil

	case "r":
		if m.secretDetails == nil {
			return m, nil
		}

		m.busy = true
		return m, m.runGetSecretDetailsCmd(m.secretDetails.ID)

	case "e":
		m.initSecretEditForm()
		return m, nil

	case "d":
		if m.secretDetails == nil {
			return m, nil
		}

		m.busy = true
		return m, m.runDeleteSecretCmd(m.secretDetails.ID)
	}

	return m, nil
}

// updateMenu обрабатывает нажатия клавиш на экране главного меню.
func (m *Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.menuIndex > 0 {
			m.menuIndex--
		}
		return m, nil

	case "down", "j":
		if m.menuIndex < len(menuItems)-1 {
			m.menuIndex++
		}
		return m, nil

	case "enter":
		return m.selectMenuItem()
	}

	return m, nil
}

// updateForm обрабатывает специальные клавиши на экране формы и сообщает, было ли событие обработано.
func (m *Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.busy {
		return m, nil, true
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit, true

	case "esc":
		m.resetToMenu()
		return m, nil, true

	case "tab", "shift+tab", "up", "down":
		m.moveFormFocus(msg.String())
		return m, nil, true

	case "enter":
		if m.focusIndex == len(m.inputs)-1 {
			m.blurInputs()
			m.busy = true

			switch m.screen {
			case screenRegister:
				return m, m.runRegisterCmd(), true
			case screenLogin:
				return m, m.runLoginCmd(), true
			case screenCreateTextSecret:
				return m, m.runCreateTextSecretCmd(), true
			case screenCreateCredentialsSecret:
				return m, m.runCreateCredentialsSecretCmd(), true
			case screenCreateCardSecret:
				return m, m.runCreateCardSecretCmd(), true
			case screenUpdateTextSecret:
				return m, m.runUpdateTextSecretCmd(), true
			case screenUpdateCredentialsSecret:
				return m, m.runUpdateCredentialsSecretCmd(), true
			case screenUpdateCardSecret:
				return m, m.runUpdateCardSecretCmd(), true
			}
		}

		m.moveFormFocus("down")
		return m, nil, true
	}

	// Обычные символы не обрабатываем здесь,
	// чтобы они попали в textinput.Update(...)
	return m, nil, false
}

// selectMenuItem запускает действие, выбранное в главном меню.
func (m *Model) selectMenuItem() (tea.Model, tea.Cmd) {
	switch menuItems[m.menuIndex] {
	case "register":
		m.initAuthForm(screenRegister)
		return m, nil

	case "login":
		m.initAuthForm(screenLogin)
		return m, nil

	case "logout":
		m.busy = true
		return m, m.runLogoutCmd()

	case "version":
		m.screen = screenMessage
		m.message = buildinfo.Current().String()
		return m, nil

	case "me":
		m.busy = true
		return m, m.runMeCmd()

	case "secrets":
		m.busy = true
		return m, m.runListSecretsCmd()

	case "add text":
		m.initSecretForm(screenCreateTextSecret)
		return m, nil

	case "add credentials":
		m.initSecretForm(screenCreateCredentialsSecret)
		return m, nil

	case "add card":
		m.initSecretForm(screenCreateCardSecret)
		return m, nil

	case "add file":
		m.initSecretForm(screenCreateBinarySecret)
		return m, nil

	case "quit":
		return m, tea.Quit
	}

	return m, nil
}

// initAuthForm подготавливает поля ввода для экрана регистрации или входа.
func (m *Model) initAuthForm(screen uiScreen) {
	m.screen = screen
	m.busy = false
	m.message = ""

	loginInput := textinput.New()
	loginInput.Placeholder = "Login"
	loginInput.Focus()
	loginInput.CharLimit = 256
	loginInput.Width = 40

	passwordInput := textinput.New()
	passwordInput.Placeholder = "Password"
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '•'
	passwordInput.CharLimit = 256
	passwordInput.Width = 40

	m.inputs = []textinput.Model{loginInput, passwordInput}
	m.focusIndex = 0
}

// initSecretForm подготавливает поля ввода для создания секрета.
func (m *Model) initSecretForm(screen uiScreen) {
	m.screen = screen
	m.busy = false
	m.message = ""

	switch screen {
	case screenCreateTextSecret:
		metaInput := textinput.New()
		metaInput.Placeholder = "Meta"
		metaInput.Focus()
		metaInput.CharLimit = 256
		metaInput.Width = 50

		textInput := textinput.New()
		textInput.Placeholder = "Text"
		textInput.CharLimit = 2048
		textInput.Width = 50

		m.inputs = []textinput.Model{metaInput, textInput}
		m.focusIndex = 0

	case screenCreateCredentialsSecret:
		metaInput := textinput.New()
		metaInput.Placeholder = "Meta"
		metaInput.Focus()
		metaInput.CharLimit = 256
		metaInput.Width = 50

		loginInput := textinput.New()
		loginInput.Placeholder = "Login"
		loginInput.CharLimit = 256
		loginInput.Width = 50

		passwordInput := textinput.New()
		passwordInput.Placeholder = "Password"
		passwordInput.EchoMode = textinput.EchoPassword
		passwordInput.EchoCharacter = '•'
		passwordInput.CharLimit = 256
		passwordInput.Width = 50

		m.inputs = []textinput.Model{metaInput, loginInput, passwordInput}
		m.focusIndex = 0

	case screenCreateCardSecret:
		metaInput := textinput.New()
		metaInput.Placeholder = "Meta"
		metaInput.Focus()
		metaInput.CharLimit = 256
		metaInput.Width = 50

		numberInput := textinput.New()
		numberInput.Placeholder = "Card number"
		numberInput.CharLimit = 32
		numberInput.Width = 50

		cardholderInput := textinput.New()
		cardholderInput.Placeholder = "Cardholder"
		cardholderInput.CharLimit = 256
		cardholderInput.Width = 50

		expiryMonthInput := textinput.New()
		expiryMonthInput.Placeholder = "Expiry month"
		expiryMonthInput.CharLimit = 2
		expiryMonthInput.Width = 50

		expiryYearInput := textinput.New()
		expiryYearInput.Placeholder = "Expiry year"
		expiryYearInput.CharLimit = 4
		expiryYearInput.Width = 50

		cvvInput := textinput.New()
		cvvInput.Placeholder = "CVV"
		cvvInput.EchoMode = textinput.EchoPassword
		cvvInput.EchoCharacter = '•'
		cvvInput.CharLimit = 4
		cvvInput.Width = 50

		m.inputs = []textinput.Model{
			metaInput,
			numberInput,
			cardholderInput,
			expiryMonthInput,
			expiryYearInput,
			cvvInput,
		}
		m.focusIndex = 0

	case screenCreateBinarySecret:
		metaInput := textinput.New()
		metaInput.Placeholder = "Meta"
		metaInput.Focus()
		metaInput.CharLimit = 256
		metaInput.Width = 50

		pathInput := textinput.New()
		pathInput.Placeholder = "File path"
		pathInput.CharLimit = 4096
		pathInput.Width = 50

		m.inputs = []textinput.Model{
			metaInput,
			pathInput,
		}
		m.focusIndex = 0

	}
}

// initSecretEditForm подготавливает форму редактирования секрета по текущим деталям.
func (m *Model) initSecretEditForm() {
	if m.secretDetails == nil {
		m.screen = screenMessage
		m.message = "No secret selected for editing"
		return
	}

	m.busy = false
	m.message = ""
	m.editingSecretID = m.secretDetails.ID

	switch m.secretDetails.Type {
	case "text":
		m.screen = screenUpdateTextSecret

		metaInput := textinput.New()
		metaInput.Placeholder = "Meta"
		metaInput.SetValue(m.secretDetails.Meta)
		metaInput.Focus()
		metaInput.CharLimit = 256
		metaInput.Width = 50

		textInput := textinput.New()
		textInput.Placeholder = "Text"
		if textValue, ok := m.secretDetails.Data["text"].(string); ok {
			textInput.SetValue(textValue)
		}
		textInput.CharLimit = 2048
		textInput.Width = 50

		m.inputs = []textinput.Model{metaInput, textInput}
		m.focusIndex = 0

	case "credentials":
		m.screen = screenUpdateCredentialsSecret

		metaInput := textinput.New()
		metaInput.Placeholder = "Meta"
		metaInput.SetValue(m.secretDetails.Meta)
		metaInput.Focus()
		metaInput.CharLimit = 256
		metaInput.Width = 50

		loginInput := textinput.New()
		loginInput.Placeholder = "Login"
		if loginValue, ok := m.secretDetails.Data["login"].(string); ok {
			loginInput.SetValue(loginValue)
		}
		loginInput.CharLimit = 256
		loginInput.Width = 50

		passwordInput := textinput.New()
		passwordInput.Placeholder = "Password"
		passwordInput.EchoMode = textinput.EchoPassword
		passwordInput.EchoCharacter = '•'
		if passwordValue, ok := m.secretDetails.Data["password"].(string); ok {
			passwordInput.SetValue(passwordValue)
		}
		passwordInput.CharLimit = 256
		passwordInput.Width = 50

		m.inputs = []textinput.Model{metaInput, loginInput, passwordInput}
		m.focusIndex = 0

	case "card":
		m.screen = screenUpdateCardSecret

		metaInput := textinput.New()
		metaInput.Placeholder = "Meta"
		metaInput.SetValue(m.secretDetails.Meta)
		metaInput.Focus()
		metaInput.CharLimit = 256
		metaInput.Width = 50

		numberInput := textinput.New()
		numberInput.Placeholder = "Card number"
		if numberValue, ok := m.secretDetails.Data["number"].(string); ok {
			numberInput.SetValue(numberValue)
		}
		numberInput.CharLimit = 32
		numberInput.Width = 50

		cardholderInput := textinput.New()
		cardholderInput.Placeholder = "Cardholder"
		if cardholderValue, ok := m.secretDetails.Data["cardholder"].(string); ok {
			cardholderInput.SetValue(cardholderValue)
		}
		cardholderInput.CharLimit = 256
		cardholderInput.Width = 50

		expiryMonthInput := textinput.New()
		expiryMonthInput.Placeholder = "Expiry month"
		if expiryMonthValue, ok := m.secretDetails.Data["expiry_month"]; ok {
			expiryMonthInput.SetValue(fmt.Sprintf("%v", expiryMonthValue))
		}
		expiryMonthInput.CharLimit = 2
		expiryMonthInput.Width = 50

		expiryYearInput := textinput.New()
		expiryYearInput.Placeholder = "Expiry year"
		if expiryYearValue, ok := m.secretDetails.Data["expiry_year"]; ok {
			expiryYearInput.SetValue(fmt.Sprintf("%v", expiryYearValue))
		}
		expiryYearInput.CharLimit = 4
		expiryYearInput.Width = 50

		cvvInput := textinput.New()
		cvvInput.Placeholder = "CVV"
		cvvInput.EchoMode = textinput.EchoPassword
		cvvInput.EchoCharacter = '•'
		if cvvValue, ok := m.secretDetails.Data["cvv"].(string); ok {
			cvvInput.SetValue(cvvValue)
		}
		cvvInput.CharLimit = 4
		cvvInput.Width = 50

		m.inputs = []textinput.Model{
			metaInput,
			numberInput,
			cardholderInput,
			expiryMonthInput,
			expiryYearInput,
			cvvInput,
		}
		m.focusIndex = 0

	default:
		m.screen = screenMessage
		m.message = "Update is supported only for text, credentials and card right now"
	}
}

// moveFormFocus переключает фокус между полями формы.
func (m *Model) moveFormFocus(direction string) {
	if len(m.inputs) == 0 {
		return
	}

	switch direction {
	case "up", "shift+tab":
		m.focusIndex--
	default:
		m.focusIndex++
	}

	if m.focusIndex >= len(m.inputs) {
		m.focusIndex = 0
	}
	if m.focusIndex < 0 {
		m.focusIndex = len(m.inputs) - 1
	}

	for i := range m.inputs {
		if i == m.focusIndex {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
}

// blurInputs снимает фокус со всех полей ввода формы.
func (m *Model) blurInputs() {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
}

// resetToMenu сбрасывает состояние формы и возвращает пользователя в главное меню.
func (m *Model) resetToMenu() {
	m.screen = screenMenu
	m.inputs = nil
	m.focusIndex = 0
	m.busy = false
}

// viewMenu формирует текстовое представление главного меню TUI.
func (m *Model) viewMenu() string {
	var b strings.Builder

	b.WriteString("GophKeeper\n\n")
	b.WriteString("Session status: " + m.currentSessionStatusLine() + "\n\n")
	b.WriteString("Choose an action:\n\n")

	for i, item := range menuItems {
		prefix := "  "
		if i == m.menuIndex {
			prefix = "> "
		}

		b.WriteString(prefix + item + "\n")
	}

	b.WriteString("\nUse ↑/↓ (or j/k), Enter to select, q to quit.\n")

	return b.String()
}

// viewSecretsList формирует текстовое представление списка секретов.
func (m *Model) viewSecretsList() string {
	var b strings.Builder

	b.WriteString("Secrets\n\n")

	if len(m.secrets) == 0 {
		b.WriteString("No secrets found.\n\n")
		b.WriteString("Esc to return, r to reload.\n")
		return b.String()
	}

	for i, item := range m.secrets {
		prefix := "  "
		if i == m.secretsIndex {
			prefix = "> "
		}

		b.WriteString(prefix + formatSecretListItem(item) + "\n")
	}

	b.WriteString("\nUse ↑/↓ (or j/k), Enter to open, r to reload, Esc to return.\n")

	return b.String()
}

// viewSecretDetails формирует текстовое представление одного секрета.
func (m *Model) viewSecretDetails() string {
	var b strings.Builder

	if m.secretDetails == nil {
		return "Secret details are not loaded.\n\nPress Enter or Esc to return.\n"
	}

	item := m.secretDetails

	b.WriteString("Secret details\n\n")
	b.WriteString("ID: " + item.ID + "\n")
	b.WriteString("Type: " + item.Type + "\n")
	b.WriteString("Meta: " + item.Meta + "\n")
	b.WriteString(fmt.Sprintf("Version: %d\n", item.Version))
	b.WriteString("CreatedAt: " + item.CreatedAt + "\n")
	b.WriteString("UpdatedAt: " + item.UpdatedAt + "\n\n")

	switch item.Type {
	case "text":
		if textValue, ok := item.Data["text"].(string); ok {
			b.WriteString("Text:\n")
			b.WriteString(textValue + "\n")
		} else {
			b.WriteString("Text:\n<invalid text payload>\n")
		}

	case "credentials":
		loginValue, _ := item.Data["login"].(string)
		passwordValue, _ := item.Data["password"].(string)

		b.WriteString("Login:\n")
		b.WriteString(loginValue + "\n\n")
		b.WriteString("Password:\n")
		b.WriteString(passwordValue + "\n")

	case "card":
		numberValue, _ := item.Data["number"].(string)
		cardholderValue, _ := item.Data["cardholder"].(string)

		b.WriteString("Card number:\n")
		b.WriteString(numberValue + "\n\n")
		b.WriteString("Cardholder:\n")
		b.WriteString(cardholderValue + "\n\n")

		if expiryMonth, ok := item.Data["expiry_month"]; ok {
			b.WriteString("Expiry month:\n")
			b.WriteString(fmt.Sprintf("%v\n\n", expiryMonth))
		}

		if expiryYear, ok := item.Data["expiry_year"]; ok {
			b.WriteString("Expiry year:\n")
			b.WriteString(fmt.Sprintf("%v\n\n", expiryYear))
		}

		if cvvValue, ok := item.Data["cvv"].(string); ok {
			b.WriteString("CVV:\n")
			b.WriteString(cvvValue + "\n")
		}

	case "binary":
		filenameValue, _ := item.Data["filename"].(string)
		mimeValue, _ := item.Data["mime_type"].(string)
		contentBase64Value, _ := item.Data["content_base64"].(string)

		b.WriteString("Filename:\n")
		b.WriteString(filenameValue + "\n\n")
		b.WriteString("MIME type:\n")
		b.WriteString(mimeValue + "\n\n")
		b.WriteString("Content length (base64 chars):\n")
		b.WriteString(fmt.Sprintf("%d\n", len(contentBase64Value)))

	default:
		b.WriteString("Payload:\n")
		b.WriteString(fmt.Sprintf("%v\n", item.Data))
	}

	b.WriteString("\nPress Enter or Esc to return, r to reload, e to edit, d to delete.\n")

	return b.String()
}

// currentSessionStatusLine возвращает краткую строку состояния текущей локальной сессии.
func (m *Model) currentSessionStatusLine() string {
	if !m.hasLocalSession {
		if strings.TrimSpace(m.sessionStatus) != "" {
			return m.sessionStatus
		}
		return "No active local session"
	}

	if !m.sessionValid {
		if strings.TrimSpace(m.sessionStatus) != "" {
			return m.sessionStatus
		}
		return "Local session found, but token is invalid"
	}

	expiresAt := ""
	if !m.sessionExpiresAt.IsZero() {
		expiresAt = m.sessionExpiresAt.Format(time.RFC3339)
	}

	if expiresAt != "" {
		return fmt.Sprintf(
			"Logged in as %s (session=%s, expires=%s)",
			m.sessionUserID,
			m.sessionSessionID,
			expiresAt,
		)
	}

	return fmt.Sprintf(
		"Logged in as %s (session=%s)",
		m.sessionUserID,
		m.sessionSessionID,
	)
}

// formatSecretListItem возвращает краткую строку представления секрета в списке.
func formatSecretListItem(item clientapi.SecretResponse) string {
	meta := strings.TrimSpace(item.Meta)
	if meta == "" {
		meta = "-"
	}

	return fmt.Sprintf("[%s] id=%s meta=%s", item.Type, item.ID, meta)
}

// viewForm формирует текстовое представление формы входа или регистрации.
func (m *Model) viewForm(title string) string {
	var b strings.Builder

	b.WriteString(title + "\n\n")

	if len(m.inputs) >= 1 {
		b.WriteString("Login:\n")
		b.WriteString(m.inputs[0].View() + "\n\n")
	}

	if len(m.inputs) >= 2 {
		b.WriteString("Password:\n")
		b.WriteString(m.inputs[1].View() + "\n\n")
	}

	if m.busy {
		b.WriteString("working...\n")
	} else {
		b.WriteString("Tab/↑/↓ to switch field, Enter to submit, Esc to cancel.\n")
	}

	return b.String()
}

// viewSecretForm формирует текстовое представление формы создания секрета.
func (m *Model) viewSecretForm(title string) string {
	var b strings.Builder

	b.WriteString(title + "\n\n")

	for _, input := range m.inputs {
		b.WriteString(input.Placeholder + ":\n")
		b.WriteString(input.View() + "\n\n")
	}

	if m.busy {
		b.WriteString("working...\n")
	} else {
		b.WriteString("Tab/↑/↓ to switch field, Enter to submit, Esc to cancel.\n")
	}

	return b.String()
}

// runAction запускает выполнение команды в режиме работы с аргументами.
func (m *Model) runAction() tea.Cmd {
	return func() tea.Msg {
		text, err := m.execute(context.Background())
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return actionResultMsg{text: text}
	}
}

// runRegisterCmd запускает регистрацию пользователя из интерактивной формы.
func (m *Model) runRegisterCmd() tea.Cmd {
	login := strings.TrimSpace(m.inputs[0].Value())
	password := m.inputs[1].Value()

	return func() tea.Msg {
		resp, err := m.api.Register(context.Background(), clientapi.RegisterRequest{
			Login:    login,
			Password: password,
		})
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return actionResultMsg{
			text: fmt.Sprintf("Registered: login=%s id=%s", resp.Login, resp.ID),
		}
	}
}

// runLoginCmd выполняет вход пользователя и сохраняет локальную сессию.
func (m *Model) runLoginCmd() tea.Cmd {
	login := strings.TrimSpace(m.inputs[0].Value())
	password := m.inputs[1].Value()

	return func() tea.Msg {
		resp, err := m.api.Login(context.Background(), clientapi.LoginRequest{
			Login:    login,
			Password: password,
		})
		if err != nil {
			return actionErrorMsg{err: err}
		}

		expiresAt, err := time.Parse(time.RFC3339, strings.TrimSpace(resp.ExpiresAt))
		if err != nil {
			return actionErrorMsg{err: fmt.Errorf("parse login expires_at: %w", err)}
		}

		meResp, err := m.api.Me(context.Background(), resp.Token)
		if err != nil {
			return actionErrorMsg{err: err}
		}

		session := local.Session{
			Token:     resp.Token,
			UserID:    meResp.UserID,
			SessionID: meResp.SessionID,
			ExpiresAt: expiresAt,
		}

		if err := m.store.SaveSession(session); err != nil {
			return actionErrorMsg{err: fmt.Errorf("save local session: %w", err)}
		}

		return sessionStatusMsg{
			hasLocalSession: true,
			sessionValid:    true,
			userID:          meResp.UserID,
			sessionID:       meResp.SessionID,
			expiresAt:       expiresAt,
			status:          "Active local session",
			showMessage: fmt.Sprintf(
				"Login successful: user_id=%s session_id=%s",
				meResp.UserID,
				meResp.SessionID,
			),
		}
	}
}

// loadSessionStatusCmd загружает локальную сессию и проверяет её валидность через API.
func (m *Model) loadSessionStatusCmd() tea.Cmd {
	return func() tea.Msg {
		session, err := m.store.LoadSession()
		if err != nil {
			if errors.Is(err, local.ErrSessionNotFound) {
				return sessionStatusMsg{
					hasLocalSession: false,
					sessionValid:    false,
					status:          "No active local session",
				}
			}

			return actionErrorMsg{err: fmt.Errorf("load local session: %w", err)}
		}

		meResp, err := m.api.Me(context.Background(), session.Token)
		if err != nil {
			return sessionStatusMsg{
				hasLocalSession: true,
				sessionValid:    false,
				expiresAt:       session.ExpiresAt,
				status:          "Local session found, but token is invalid on server",
			}
		}

		return sessionStatusMsg{
			hasLocalSession: true,
			sessionValid:    true,
			userID:          meResp.UserID,
			sessionID:       meResp.SessionID,
			expiresAt:       session.ExpiresAt,
			status:          "Active local session",
		}
	}
}

// runMeCmd проверяет сохранённую локальную сессию и показывает текущего пользователя.
func (m *Model) runMeCmd() tea.Cmd {
	return func() tea.Msg {
		session, err := m.store.LoadSession()
		if err != nil {
			if errors.Is(err, local.ErrSessionNotFound) {
				return sessionStatusMsg{
					hasLocalSession: false,
					sessionValid:    false,
					status:          "No active local session",
					showMessage:     "No active local session",
				}
			}

			return actionErrorMsg{err: fmt.Errorf("load local session: %w", err)}
		}

		meResp, err := m.api.Me(context.Background(), session.Token)
		if err != nil {
			return sessionStatusMsg{
				hasLocalSession: true,
				sessionValid:    false,
				expiresAt:       session.ExpiresAt,
				status:          "Local session found, but token is invalid on server",
				showMessage:     "Saved session exists, but token is invalid on server",
			}
		}

		return sessionStatusMsg{
			hasLocalSession: true,
			sessionValid:    true,
			userID:          meResp.UserID,
			sessionID:       meResp.SessionID,
			expiresAt:       session.ExpiresAt,
			status:          "Active local session",
			showMessage: fmt.Sprintf(
				"Current user: %s\nSession ID: %s\nExpires at: %s",
				meResp.UserID,
				meResp.SessionID,
				session.ExpiresAt.Format(time.RFC3339),
			),
		}
	}
}

// runListSecretsCmd загружает список секретов текущего пользователя.
func (m *Model) runListSecretsCmd() tea.Cmd {
	return func() tea.Msg {
		session, err := m.store.LoadSession()
		if err != nil {
			if errors.Is(err, local.ErrSessionNotFound) {
				return actionErrorMsg{err: fmt.Errorf("no active local session")}
			}

			return actionErrorMsg{err: fmt.Errorf("load local session: %w", err)}
		}

		resp, err := m.api.ListSecrets(context.Background(), session.Token)
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return secretsListMsg{items: resp.Items}
	}
}

// runGetSecretDetailsCmd загружает один секрет по идентификатору.
func (m *Model) runGetSecretDetailsCmd(secretID string) tea.Cmd {
	return func() tea.Msg {
		session, err := m.store.LoadSession()
		if err != nil {
			if errors.Is(err, local.ErrSessionNotFound) {
				return actionErrorMsg{err: fmt.Errorf("no active local session")}
			}

			return actionErrorMsg{err: fmt.Errorf("load local session: %w", err)}
		}

		resp, err := m.api.GetSecretByID(context.Background(), session.Token, secretID)
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return secretDetailsMsg{item: *resp}
	}
}

// runCreateTextSecretCmd создаёт текстовый секрет через интерактивную форму.
func (m *Model) runCreateTextSecretCmd() tea.Cmd {
	meta := strings.TrimSpace(m.inputs[0].Value())
	text := strings.TrimSpace(m.inputs[1].Value())

	return func() tea.Msg {
		session, err := m.store.LoadSession()
		if err != nil {
			if errors.Is(err, local.ErrSessionNotFound) {
				return actionErrorMsg{err: fmt.Errorf("no active local session")}
			}

			return actionErrorMsg{err: fmt.Errorf("load local session: %w", err)}
		}

		resp, err := m.api.CreateSecret(context.Background(), session.Token, clientapi.SecretUpsertRequest{
			Type: "text",
			Meta: meta,
			Data: clientapi.TextData{
				Text: text,
			},
		})
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return actionResultMsg{
			text: fmt.Sprintf("Text secret created: id=%s type=%s", resp.ID, resp.Type),
		}
	}
}

// runUpdateTextSecretCmd обновляет текстовый секрет через интерактивную форму.
func (m *Model) runUpdateTextSecretCmd() tea.Cmd {
	meta := strings.TrimSpace(m.inputs[0].Value())
	text := strings.TrimSpace(m.inputs[1].Value())
	secretID := strings.TrimSpace(m.editingSecretID)

	return func() tea.Msg {
		session, err := m.store.LoadSession()
		if err != nil {
			if errors.Is(err, local.ErrSessionNotFound) {
				return actionErrorMsg{err: fmt.Errorf("no active local session")}
			}

			return actionErrorMsg{err: fmt.Errorf("load local session: %w", err)}
		}

		resp, err := m.api.UpdateSecret(context.Background(), session.Token, secretID, clientapi.SecretUpsertRequest{
			Type: "text",
			Meta: meta,
			Data: clientapi.TextData{
				Text: text,
			},
		})
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return secretDetailsMsg{item: *resp}
	}
}

// runUpdateCredentialsSecretCmd обновляет секрет типа credentials через интерактивную форму.
func (m *Model) runUpdateCredentialsSecretCmd() tea.Cmd {
	meta := strings.TrimSpace(m.inputs[0].Value())
	login := strings.TrimSpace(m.inputs[1].Value())
	password := m.inputs[2].Value()
	secretID := strings.TrimSpace(m.editingSecretID)

	return func() tea.Msg {
		session, err := m.store.LoadSession()
		if err != nil {
			if errors.Is(err, local.ErrSessionNotFound) {
				return actionErrorMsg{err: fmt.Errorf("no active local session")}
			}

			return actionErrorMsg{err: fmt.Errorf("load local session: %w", err)}
		}

		resp, err := m.api.UpdateSecret(context.Background(), session.Token, secretID, clientapi.SecretUpsertRequest{
			Type: "credentials",
			Meta: meta,
			Data: clientapi.CredentialsData{
				Login:    login,
				Password: password,
			},
		})
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return secretDetailsMsg{item: *resp}
	}
}

// runCreateCredentialsSecretCmd создаёт секрет типа credentials через интерактивную форму.
func (m *Model) runCreateCredentialsSecretCmd() tea.Cmd {
	meta := strings.TrimSpace(m.inputs[0].Value())
	login := strings.TrimSpace(m.inputs[1].Value())
	password := m.inputs[2].Value()

	return func() tea.Msg {
		session, err := m.store.LoadSession()
		if err != nil {
			if errors.Is(err, local.ErrSessionNotFound) {
				return actionErrorMsg{err: fmt.Errorf("no active local session")}
			}

			return actionErrorMsg{err: fmt.Errorf("load local session: %w", err)}
		}

		resp, err := m.api.CreateSecret(context.Background(), session.Token, clientapi.SecretUpsertRequest{
			Type: "credentials",
			Meta: meta,
			Data: clientapi.CredentialsData{
				Login:    login,
				Password: password,
			},
		})
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return actionResultMsg{
			text: fmt.Sprintf("Credentials secret created: id=%s type=%s", resp.ID, resp.Type),
		}
	}
}

// runCreateCardSecretCmd создаёт секрет типа card через интерактивную форму.
func (m *Model) runCreateCardSecretCmd() tea.Cmd {
	meta := strings.TrimSpace(m.inputs[0].Value())
	number := strings.TrimSpace(m.inputs[1].Value())
	cardholder := strings.TrimSpace(m.inputs[2].Value())
	expiryMonthRaw := strings.TrimSpace(m.inputs[3].Value())
	expiryYearRaw := strings.TrimSpace(m.inputs[4].Value())
	cvv := strings.TrimSpace(m.inputs[5].Value())

	return func() tea.Msg {
		session, err := m.store.LoadSession()
		if err != nil {
			if errors.Is(err, local.ErrSessionNotFound) {
				return actionErrorMsg{err: fmt.Errorf("no active local session")}
			}

			return actionErrorMsg{err: fmt.Errorf("load local session: %w", err)}
		}

		expiryMonth, err := strconv.ParseUint(expiryMonthRaw, 10, 8)
		if err != nil {
			return actionErrorMsg{err: fmt.Errorf("invalid expiry month: %w", err)}
		}

		expiryYear, err := strconv.ParseUint(expiryYearRaw, 10, 16)
		if err != nil {
			return actionErrorMsg{err: fmt.Errorf("invalid expiry year: %w", err)}
		}

		if _, err := m.api.CreateSecret(context.Background(), session.Token, clientapi.SecretUpsertRequest{
			Type: "card",
			Meta: meta,
			Data: clientapi.CardData{
				Number:      number,
				Cardholder:  cardholder,
				ExpiryMonth: uint8(expiryMonth),
				ExpiryYear:  uint16(expiryYear),
				CVV:         cvv,
			},
		}); err != nil {
			return actionErrorMsg{err: err}
		}

		// После создания сразу перечитываем список,
		// чтобы новая карта появилась в экране secrets.
		resp, err := m.api.ListSecrets(context.Background(), session.Token)
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return secretsListMsg{items: resp.Items}
	}
}

// runUpdateCardSecretCmd обновляет секрет типа card через интерактивную форму.
func (m *Model) runUpdateCardSecretCmd() tea.Cmd {
	meta := strings.TrimSpace(m.inputs[0].Value())
	number := strings.TrimSpace(m.inputs[1].Value())
	cardholder := strings.TrimSpace(m.inputs[2].Value())
	expiryMonthRaw := strings.TrimSpace(m.inputs[3].Value())
	expiryYearRaw := strings.TrimSpace(m.inputs[4].Value())
	cvv := strings.TrimSpace(m.inputs[5].Value())
	secretID := strings.TrimSpace(m.editingSecretID)

	return func() tea.Msg {
		session, err := m.store.LoadSession()
		if err != nil {
			if errors.Is(err, local.ErrSessionNotFound) {
				return actionErrorMsg{err: fmt.Errorf("no active local session")}
			}

			return actionErrorMsg{err: fmt.Errorf("load local session: %w", err)}
		}

		expiryMonth, err := strconv.ParseUint(expiryMonthRaw, 10, 8)
		if err != nil {
			return actionErrorMsg{err: fmt.Errorf("invalid expiry month: %w", err)}
		}

		expiryYear, err := strconv.ParseUint(expiryYearRaw, 10, 16)
		if err != nil {
			return actionErrorMsg{err: fmt.Errorf("invalid expiry year: %w", err)}
		}

		resp, err := m.api.UpdateSecret(context.Background(), session.Token, secretID, clientapi.SecretUpsertRequest{
			Type: "card",
			Meta: meta,
			Data: clientapi.CardData{
				Number:      number,
				Cardholder:  cardholder,
				ExpiryMonth: uint8(expiryMonth),
				ExpiryYear:  uint16(expiryYear),
				CVV:         cvv,
			},
		})
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return secretDetailsMsg{item: *resp}
	}
}

// runCreateBinarySecretCmd создаёт секрет типа binary через интерактивную форму.
func (m *Model) runCreateBinarySecretCmd() tea.Cmd {
	meta := strings.TrimSpace(m.inputs[0].Value())
	filePath := strings.TrimSpace(m.inputs[1].Value())

	return func() tea.Msg {
		if filePath == "" {
			return actionErrorMsg{err: fmt.Errorf("file path is empty")}
		}

		session, err := m.store.LoadSession()
		if err != nil {
			if errors.Is(err, local.ErrSessionNotFound) {
				return actionErrorMsg{err: fmt.Errorf("no active local session")}
			}

			return actionErrorMsg{err: fmt.Errorf("load local session: %w", err)}
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return actionErrorMsg{err: fmt.Errorf("read file: %w", err)}
		}

		filename := filepath.Base(filePath)
		mimeType := mime.TypeByExtension(filepath.Ext(filename))
		if strings.TrimSpace(mimeType) == "" {
			mimeType = "application/octet-stream"
		}

		if _, err := m.api.CreateSecret(context.Background(), session.Token, clientapi.SecretUpsertRequest{
			Type: "binary",
			Meta: meta,
			Data: clientapi.BinaryData{
				Filename:      filename,
				MIMEType:      mimeType,
				ContentBase64: encodeBase64(content),
			},
		}); err != nil {
			return actionErrorMsg{err: err}
		}

		resp, err := m.api.ListSecrets(context.Background(), session.Token)
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return secretsListMsg{items: resp.Items}
	}
}

// runDeleteSecretCmd удаляет секрет пользователя по идентификатору.
func (m *Model) runDeleteSecretCmd(secretID string) tea.Cmd {
	return func() tea.Msg {
		session, err := m.store.LoadSession()
		if err != nil {
			if errors.Is(err, local.ErrSessionNotFound) {
				return actionErrorMsg{err: fmt.Errorf("no active local session")}
			}

			return actionErrorMsg{err: fmt.Errorf("load local session: %w", err)}
		}

		if err := m.api.DeleteSecret(context.Background(), session.Token, secretID); err != nil {
			return actionErrorMsg{err: err}
		}

		// После удаления сразу перечитываем список.
		resp, err := m.api.ListSecrets(context.Background(), session.Token)
		if err != nil {
			return actionErrorMsg{err: err}
		}

		return secretsListMsg{items: resp.Items}
	}
}

// runLogoutCmd очищает локальную сессию пользователя.
func (m *Model) runLogoutCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.store.ClearSession(); err != nil {
			return actionErrorMsg{err: fmt.Errorf("clear local session: %w", err)}
		}

		m.secrets = nil
		m.secretsIndex = 0
		m.secretDetails = nil
		m.editingSecretID = ""

		return sessionStatusMsg{
			hasLocalSession: false,
			sessionValid:    false,
			status:          "No active local session",
			showMessage:     "Logout successful",
		}
	}
}

// execute выполняет клиентскую команду в режиме запуска с аргументами командной строки.
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

// encodeBase64 кодирует бинарные данные в base64-строку.
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
