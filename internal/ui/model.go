// Package ui implements Bluff's terminal interface using the Charm ecosystem.
package ui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/thsnkhn/bluff/internal/api"
	"github.com/thsnkhn/bluff/internal/credentials"
)

const operationTimeout = 15 * time.Second

type screen int

const (
	bootScreen screen = iota
	loginScreen
	dashboardScreen
)

// BuildInfo describes the running client build.
type BuildInfo struct {
	Version string
}

// API captures the authenticated operations used by the terminal client.
type API interface {
	Login(context.Context, string, string) (api.Session, error)
	Me(context.Context, string) (api.User, error)
	Bootstrap(context.Context, string) (api.Bootstrap, error)
	Logout(context.Context, string) error
}

// CredentialStore persists the bearer token outside the application files.
type CredentialStore interface {
	Load(context.Context) (string, error)
	Save(context.Context, string) error
	Delete(context.Context) error
}

// Model is the root Bubble Tea model.
type Model struct {
	api       API
	store     CredentialStore
	build     BuildInfo
	screen    screen
	width     int
	height    int
	spinner   spinner.Model
	form      *huh.Form
	username  string
	password  string
	loading   bool
	status    string
	err       error
	token     string
	user      api.User
	bootstrap api.Bootstrap
}

// New constructs the Bluff terminal application.
func New(client API, store CredentialStore, build BuildInfo) Model {
	busy := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	busy.Style = lipgloss.NewStyle().Foreground(colorGold)
	model := Model{
		api:     client,
		store:   store,
		build:   build,
		screen:  bootScreen,
		spinner: busy,
		loading: true,
		status:  "Opening the table",
	}
	model.resetLoginForm()
	return model
}

// Init restores any saved session while starting the loading animation.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.restoreSessionCmd())
}

// Update advances the application in response to terminal and network events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resizeForm()
		return m, nil
	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.screen == dashboardScreen && !m.loading {
			switch key {
			case "q":
				return m, tea.Quit
			case "r":
				m.loading, m.status, m.err = true, "Refreshing the ledger", nil
				return m, tea.Batch(m.spinner.Tick, m.refreshCmd())
			case "l":
				m.loading, m.status, m.err = true, "Closing your session", nil
				return m, tea.Batch(m.spinner.Tick, m.logoutCmd())
			}
		}
	}

	switch msg := msg.(type) {
	case sessionRestoredMsg:
		m.loading = false
		m.token, m.user, m.bootstrap = msg.token, msg.user, msg.bootstrap
		m.screen, m.err = dashboardScreen, nil
		return m, nil
	case loginRequiredMsg:
		m.loading, m.screen, m.err = false, loginScreen, msg.err
		m.resetLoginForm()
		m.resizeForm()
		return m, m.form.Init()
	case loginSucceededMsg:
		m.loading = false
		m.token, m.user, m.bootstrap = msg.token, msg.user, msg.bootstrap
		m.screen, m.err = dashboardScreen, nil
		m.password = ""
		return m, nil
	case operationFailedMsg:
		m.loading, m.err = false, msg.err
		if m.screen == loginScreen {
			m.password = ""
			m.resetLoginForm()
			m.resizeForm()
			return m, m.form.Init()
		}
		return m, nil
	case refreshedMsg:
		m.loading, m.bootstrap, m.err = false, msg.bootstrap, nil
		return m, nil
	case loggedOutMsg:
		m.loading, m.token, m.user, m.bootstrap = false, "", api.User{}, api.Bootstrap{}
		m.screen, m.err = loginScreen, msg.err
		m.resetLoginForm()
		m.resizeForm()
		return m, m.form.Init()
	case spinner.TickMsg:
		if m.loading || m.screen == bootScreen {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	if m.screen == loginScreen && !m.loading {
		updated, cmd := m.form.Update(msg)
		if form, ok := updated.(*huh.Form); ok {
			m.form = form
		}
		if m.form.State == huh.StateCompleted {
			m.loading, m.status, m.err = true, "Taking your seat", nil
			return m, tea.Batch(m.spinner.Tick, m.loginCmd(m.username, m.password))
		}
		if m.form.State == huh.StateAborted {
			return m, tea.Quit
		}
		return m, cmd
	}

	return m, nil
}

// View renders the current full-screen application.
func (m Model) View() tea.View {
	var content string
	switch m.screen {
	case bootScreen:
		content = m.loadingView()
	case loginScreen:
		content = m.loginView()
	case dashboardScreen:
		content = m.dashboardView()
	}
	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "Bluff"
	return view
}

func (m *Model) resetLoginForm() {
	m.form = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Username").Description("Your table username").Value(&m.username).
			Placeholder("bluff").Validate(required("Enter your username")),
		huh.NewInput().Title("Password").Description("Stored only in your system keychain").Value(&m.password).
			EchoMode(huh.EchoModePassword).Validate(required("Enter your password")),
	)).WithTheme(huh.ThemeFunc(bluffTheme)).WithShowHelp(true)
}

func (m *Model) resizeForm() {
	if m.form == nil {
		return
	}
	width := min(max(m.width-12, 32), 56)
	m.form.WithWidth(width).WithHeight(min(max(m.height-14, 8), 16))
}

func required(message string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return errors.New(message)
		}
		return nil
	}
}

type sessionRestoredMsg struct {
	token     string
	user      api.User
	bootstrap api.Bootstrap
}

type loginRequiredMsg struct{ err error }
type loginSucceededMsg sessionRestoredMsg
type operationFailedMsg struct{ err error }
type refreshedMsg struct{ bootstrap api.Bootstrap }
type loggedOutMsg struct{ err error }

func (m Model) restoreSessionCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		token, err := m.store.Load(ctx)
		if errors.Is(err, credentials.ErrNotFound) {
			return loginRequiredMsg{}
		}
		if err != nil {
			return loginRequiredMsg{err: fmt.Errorf("could not open the system keychain: %w", err)}
		}
		user, err := m.api.Me(ctx, token)
		if err != nil {
			if api.IsUnauthorized(err) {
				_ = m.store.Delete(ctx)
				return loginRequiredMsg{err: errors.New("your saved session expired; sign in again")}
			}
			return loginRequiredMsg{err: err}
		}
		bootstrap, err := m.api.Bootstrap(ctx, token)
		if err != nil {
			return loginRequiredMsg{err: err}
		}
		return sessionRestoredMsg{token: token, user: user, bootstrap: bootstrap}
	}
}

func (m Model) loginCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		session, err := m.api.Login(ctx, username, password)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		if err := m.store.Save(ctx, session.Token); err != nil {
			_ = m.api.Logout(ctx, session.Token)
			return operationFailedMsg{err: fmt.Errorf("could not save the session securely: %w", err)}
		}
		bootstrap, err := m.api.Bootstrap(ctx, session.Token)
		if err != nil {
			_ = m.store.Delete(ctx)
			return operationFailedMsg{err: err}
		}
		return loginSucceededMsg{token: session.Token, user: session.User, bootstrap: bootstrap}
	}
}

func (m Model) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		bootstrap, err := m.api.Bootstrap(ctx, m.token)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return refreshedMsg{bootstrap: bootstrap}
	}
}

func (m Model) logoutCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		remoteErr := m.api.Logout(ctx, m.token)
		localErr := m.store.Delete(ctx)
		if localErr != nil {
			return loggedOutMsg{err: fmt.Errorf("server session closed, but the keychain entry could not be removed: %w", localErr)}
		}
		if remoteErr != nil && !api.IsUnauthorized(remoteErr) {
			return loggedOutMsg{err: errors.New("signed out on this device; the server could not confirm logout")}
		}
		return loggedOutMsg{}
	}
}

func sortedPlayers(players []api.Player) []api.Player {
	result := append([]api.Player(nil), players...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Standing == result[j].Standing {
			return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
		}
		return result[i].Standing > result[j].Standing
	})
	return result
}
