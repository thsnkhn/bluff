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
	homeScreen
	loginScreen
	aboutScreen
	appMenuScreen
	usersScreen
	createUserScreen
	dashboardScreen
)

const (
	homeSignIn = iota
	homeCheckConnection
	homeAbout
	homeQuit
	homeItemCount
)

// BuildInfo describes the running client build.
type BuildInfo struct {
	Version string
}

type loginValues struct {
	username string
	password string
}

type createUserValues struct {
	username string
	password string
	role     string
}

type appAction string

const (
	actionUsers       appAction = "users"
	actionTables      appAction = "tables"
	actionGames       appAction = "games"
	actionGameFormats appAction = "game-formats"
	actionMyInfo      appAction = "my-info"
	actionLogout      appAction = "logout"
	actionQuit        appAction = "quit"
)

type appMenuItem struct {
	title   string
	action  appAction
	enabled bool
}

// API captures the authenticated operations used by the terminal client.
type API interface {
	Health(context.Context) error
	Login(context.Context, string, string) (api.Session, error)
	Me(context.Context, string) (api.User, error)
	Bootstrap(context.Context, string) (api.Bootstrap, error)
	Users(context.Context, string) ([]api.User, error)
	CreateUser(context.Context, string, string, string, string) (api.User, error)
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
	api                API
	store              CredentialStore
	build              BuildInfo
	screen             screen
	width              int
	height             int
	spinner            spinner.Model
	form               *huh.Form
	login              *loginValues
	createUser         *createUserValues
	loading            bool
	status             string
	err                error
	token              string
	user               api.User
	bootstrap          api.Bootstrap
	homeIndex          int
	appIndex           int
	usersIndex         int
	users              []api.User
	notice             string
	connected          bool
	checkingConnection bool
}

// New constructs the Bluff terminal application.
func New(client API, store CredentialStore, build BuildInfo) Model {
	busy := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	busy.Style = lipgloss.NewStyle().Foreground(colorFuchsia)
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
	return tea.Batch(tea.RequestBackgroundColor, m.spinner.Tick, m.restoreSessionCmd())
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
		if m.screen == homeScreen && !m.loading {
			switch key {
			case "up", "k", "shift+tab":
				m.homeIndex = (m.homeIndex - 1 + homeItemCount) % homeItemCount
				return m, nil
			case "down", "j", "tab":
				m.homeIndex = (m.homeIndex + 1) % homeItemCount
				return m, nil
			case "enter", " ":
				return m.activateHomeItem()
			case "q", "esc":
				return m, tea.Quit
			}
		}
		if m.screen == aboutScreen {
			if key == "esc" || key == "enter" || key == "q" {
				m.screen, m.err = homeScreen, nil
				return m, nil
			}
		}
		if m.screen == loginScreen && !m.loading && key == "esc" {
			m.screen, m.err = homeScreen, nil
			m.resetLoginForm()
			m.resizeForm()
			return m, nil
		}
		if m.screen == appMenuScreen && !m.loading {
			switch key {
			case "up", "k", "shift+tab":
				m.appIndex = m.nextAppIndex(-1)
				return m, nil
			case "down", "j", "tab":
				m.appIndex = m.nextAppIndex(1)
				return m, nil
			case "enter", " ":
				return m.activateAppItem()
			case "q":
				return m, tea.Quit
			}
		}
		if m.screen == usersScreen && !m.loading {
			switch key {
			case "up", "k":
				m.moveUsers(-1)
				return m, nil
			case "down", "j":
				m.moveUsers(1)
				return m, nil
			case "c":
				return m.openCreateUser()
			case "r":
				m.loading, m.status, m.err, m.notice = true, "Refreshing users", nil, ""
				return m, tea.Batch(m.spinner.Tick, m.loadUsersCmd())
			case "esc", "backspace":
				m.screen, m.err, m.notice = appMenuScreen, nil, ""
				return m, nil
			}
		}
		if m.screen == createUserScreen && !m.loading && key == "esc" {
			m.screen, m.err = usersScreen, nil
			m.resetCreateUserForm()
			m.resizeForm()
			return m, nil
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
	case menuMouseMsg:
		if m.screen == homeScreen && !m.loading && msg.index >= 0 && msg.index < homeItemCount {
			m.homeIndex = msg.index
			if msg.activate {
				return m.activateHomeItem()
			}
		}
		return m, nil
	case appMenuMouseMsg:
		if m.screen == appMenuScreen && !m.loading {
			items := m.appMenuItems()
			if msg.index >= 0 && msg.index < len(items) {
				m.appIndex = msg.index
				if msg.activate && items[msg.index].enabled {
					return m.activateAppItem()
				}
			}
		}
		return m, nil
	case usersMouseMsg:
		if m.screen != usersScreen || m.loading {
			return m, nil
		}
		if msg.userIndex >= 0 && msg.userIndex < len(m.users) {
			m.usersIndex = msg.userIndex
			return m, nil
		}
		if !msg.activate {
			return m, nil
		}
		switch msg.action {
		case "create":
			return m.openCreateUser()
		case "refresh":
			m.loading, m.status, m.err, m.notice = true, "Refreshing users", nil, ""
			return m, tea.Batch(m.spinner.Tick, m.loadUsersCmd())
		case "back":
			m.screen, m.err, m.notice = appMenuScreen, nil, ""
			return m, nil
		}
		return m, nil
	case dashboardMouseMsg:
		if m.screen != dashboardScreen || m.loading || !msg.activate {
			return m, nil
		}
		switch msg.action {
		case "refresh":
			m.loading, m.status, m.err = true, "Refreshing the ledger", nil
			return m, tea.Batch(m.spinner.Tick, m.refreshCmd())
		case "logout":
			m.loading, m.status, m.err = true, "Closing your session", nil
			return m, tea.Batch(m.spinner.Tick, m.logoutCmd())
		case "quit":
			return m, tea.Quit
		}
	case sessionRestoredMsg:
		m.loading = false
		m.token, m.user, m.bootstrap = msg.token, msg.user, msg.bootstrap
		m.screen, m.err, m.connected = appMenuScreen, nil, true
		m.appIndex = m.firstEnabledAppIndex()
		return m, nil
	case loginRequiredMsg:
		m.loading, m.screen, m.err = false, homeScreen, msg.err
		m.connected = msg.connected
		m.resetLoginForm()
		m.resizeForm()
		return m, nil
	case connectionCheckedMsg:
		m.loading, m.checkingConnection = false, false
		m.connected, m.err = msg.err == nil, msg.err
		return m, nil
	case loginSucceededMsg:
		m.loading = false
		m.token, m.user, m.bootstrap = msg.token, msg.user, msg.bootstrap
		m.screen, m.err, m.connected = appMenuScreen, nil, true
		m.appIndex = m.firstEnabledAppIndex()
		m.login.password = ""
		return m, nil
	case operationFailedMsg:
		m.loading, m.err = false, msg.err
		if m.screen == loginScreen {
			m.login.password = ""
			m.resetLoginForm()
			m.resizeForm()
			return m, m.form.Init()
		}
		if m.screen == createUserScreen {
			m.createUser.password = ""
			m.resetCreateUserForm()
			m.resizeForm()
			return m, m.form.Init()
		}
		return m, nil
	case refreshedMsg:
		m.loading, m.bootstrap, m.err = false, msg.bootstrap, nil
		return m, nil
	case usersLoadedMsg:
		m.loading, m.users, m.err = false, msg.users, nil
		if len(m.users) == 0 {
			m.usersIndex = 0
		} else if m.usersIndex >= len(m.users) {
			m.usersIndex = len(m.users) - 1
		}
		return m, nil
	case userCreatedMsg:
		m.loading, m.users, m.err = false, msg.users, nil
		m.screen, m.notice = usersScreen, "@"+msg.user.Username+" was added"
		for index, user := range m.users {
			if user.ID == msg.user.ID {
				m.usersIndex = index
				break
			}
		}
		m.resetCreateUserForm()
		return m, nil
	case loggedOutMsg:
		m.loading, m.token, m.user, m.bootstrap = false, "", api.User{}, api.Bootstrap{}
		m.screen, m.err = homeScreen, msg.err
		m.resetLoginForm()
		m.resizeForm()
		return m, nil
	case spinner.TickMsg:
		if m.loading || m.screen == bootScreen {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	if (m.screen == loginScreen || m.screen == createUserScreen) && !m.loading {
		updated, cmd := m.form.Update(msg)
		if form, ok := updated.(*huh.Form); ok {
			m.form = form
		}
		if m.form.State == huh.StateCompleted {
			if m.screen == createUserScreen {
				m.loading, m.status, m.err = true, "Creating user", nil
				return m, tea.Batch(m.spinner.Tick, m.createUserCmd())
			}
			m.loading, m.status, m.err = true, "Taking your seat", nil
			return m, tea.Batch(m.spinner.Tick, m.loginCmd(m.login.username, m.login.password))
		}
		if m.form.State == huh.StateAborted {
			if m.screen == createUserScreen {
				m.screen = usersScreen
			} else {
				m.screen = homeScreen
			}
			return m, nil
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
	case homeScreen:
		content = m.homeView()
	case loginScreen:
		content = m.loginView()
	case aboutScreen:
		content = m.aboutView()
	case appMenuScreen:
		content = m.appMenuView()
	case usersScreen:
		content = m.usersView()
	case createUserScreen:
		content = m.createUserView()
	case dashboardScreen:
		content = m.dashboardView()
	}
	view := tea.NewView(content)
	view.AltScreen = true
	view.MouseMode = tea.MouseModeAllMotion
	view.WindowTitle = "Bluff"
	view.OnMouse = m.mouseHandler()
	return view
}

func (m *Model) resetLoginForm() {
	username := ""
	if m.login != nil {
		username = m.login.username
	}
	m.login = &loginValues{username: username}
	m.form = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Username").Description("Your table username").Value(&m.login.username).
			Placeholder("bluff").Validate(required("Enter your username")),
		huh.NewInput().Title("Password").Description("Your password is never stored").Value(&m.login.password).
			EchoMode(huh.EchoModePassword).Validate(required("Enter your password")),
	)).WithTheme(huh.ThemeFunc(huh.ThemeCharm)).WithShowHelp(true)
}

func (m *Model) resetCreateUserForm() {
	role := "member"
	username := ""
	if m.createUser != nil {
		username = m.createUser.username
		if m.createUser.role != "" {
			role = m.createUser.role
		}
	}
	m.createUser = &createUserValues{username: username, role: role}
	m.form = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Username").Description("3-32 letters, numbers, dots, dashes, or underscores").Value(&m.createUser.username).
			Placeholder("table-friend").Validate(required("Enter a username")),
		huh.NewInput().Title("Password").Description("At least 12 characters").Value(&m.createUser.password).
			EchoMode(huh.EchoModePassword).Validate(required("Enter a password")),
		huh.NewSelect[string]().Title("Role").Description("Members can sign in; admins can manage users").
			Options(huh.NewOption("Member", "member"), huh.NewOption("Admin", "admin")).Value(&m.createUser.role),
	)).WithTheme(huh.ThemeFunc(huh.ThemeCharm)).WithShowHelp(true)
}

func (m *Model) resizeForm() {
	if m.form == nil {
		return
	}
	width := min(max(m.width-16, 32), 54)
	height := 16
	if m.screen == createUserScreen {
		height = 24
	}
	m.form.WithWidth(width).WithHeight(min(max(m.height-18, 8), height))
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

type loginRequiredMsg struct {
	err       error
	connected bool
}
type connectionCheckedMsg struct{ err error }
type loginSucceededMsg sessionRestoredMsg
type operationFailedMsg struct{ err error }
type refreshedMsg struct{ bootstrap api.Bootstrap }
type usersLoadedMsg struct{ users []api.User }
type userCreatedMsg struct {
	user  api.User
	users []api.User
}
type loggedOutMsg struct{ err error }

func (m Model) restoreSessionCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		connectionErr := m.api.Health(ctx)
		token, err := m.store.Load(ctx)
		if errors.Is(err, credentials.ErrNotFound) {
			return loginRequiredMsg{err: connectionErr, connected: connectionErr == nil}
		}
		if err != nil {
			return loginRequiredMsg{err: fmt.Errorf("could not open the system keychain: %w", err), connected: connectionErr == nil}
		}
		user, err := m.api.Me(ctx, token)
		if err != nil {
			if api.IsUnauthorized(err) {
				_ = m.store.Delete(ctx)
				return loginRequiredMsg{err: errors.New("your saved session expired; sign in again"), connected: connectionErr == nil}
			}
			return loginRequiredMsg{err: err, connected: connectionErr == nil}
		}
		bootstrap, err := m.api.Bootstrap(ctx, token)
		if err != nil {
			return loginRequiredMsg{err: err, connected: connectionErr == nil}
		}
		return sessionRestoredMsg{token: token, user: user, bootstrap: bootstrap}
	}
}

func (m Model) checkConnectionCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		return connectionCheckedMsg{err: m.api.Health(ctx)}
	}
}

func (m Model) activateHomeItem() (tea.Model, tea.Cmd) {
	switch m.homeIndex {
	case homeSignIn:
		m.screen, m.err = loginScreen, nil
		m.resetLoginForm()
		m.resizeForm()
		return m, m.form.Init()
	case homeCheckConnection:
		m.loading, m.checkingConnection, m.status, m.err = true, true, "Checking the connection", nil
		return m, tea.Batch(m.spinner.Tick, m.checkConnectionCmd())
	case homeAbout:
		m.screen, m.err = aboutScreen, nil
		return m, nil
	case homeQuit:
		return m, tea.Quit
	default:
		return m, nil
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

func (m Model) loadUsersCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		users, err := m.api.Users(ctx, m.token)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return usersLoadedMsg{users: users}
	}
}

func (m Model) createUserCmd() tea.Cmd {
	username, password, role := m.createUser.username, m.createUser.password, m.createUser.role
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		user, err := m.api.CreateUser(ctx, m.token, username, password, role)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		users, err := m.api.Users(ctx, m.token)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return userCreatedMsg{user: user, users: users}
	}
}

func (m Model) appMenuItems() []appMenuItem {
	items := make([]appMenuItem, 0, 7)
	if m.user.Role == "admin" {
		items = append(items, appMenuItem{title: "Users", action: actionUsers, enabled: true})
	}
	items = append(items,
		appMenuItem{title: "Tables", action: actionTables},
		appMenuItem{title: "Games", action: actionGames},
		appMenuItem{title: "Game formats", action: actionGameFormats},
		appMenuItem{title: "My info", action: actionMyInfo},
		appMenuItem{title: "Log out", action: actionLogout, enabled: true},
		appMenuItem{title: "Quit", action: actionQuit, enabled: true},
	)
	return items
}

func (m Model) firstEnabledAppIndex() int {
	for index, item := range m.appMenuItems() {
		if item.enabled {
			return index
		}
	}
	return 0
}

func (m Model) nextAppIndex(direction int) int {
	items := m.appMenuItems()
	if len(items) == 0 {
		return 0
	}
	index := m.appIndex
	for range items {
		index = (index + direction + len(items)) % len(items)
		if items[index].enabled {
			return index
		}
	}
	return m.appIndex
}

func (m Model) activateAppItem() (tea.Model, tea.Cmd) {
	items := m.appMenuItems()
	if m.appIndex < 0 || m.appIndex >= len(items) || !items[m.appIndex].enabled {
		return m, nil
	}
	switch items[m.appIndex].action {
	case actionUsers:
		m.screen, m.loading, m.status, m.err, m.notice = usersScreen, true, "Loading users", nil, ""
		return m, tea.Batch(m.spinner.Tick, m.loadUsersCmd())
	case actionLogout:
		m.loading, m.status, m.err = true, "Closing your session", nil
		return m, tea.Batch(m.spinner.Tick, m.logoutCmd())
	case actionQuit:
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m *Model) moveUsers(direction int) {
	if len(m.users) == 0 {
		m.usersIndex = 0
		return
	}
	m.usersIndex = (m.usersIndex + direction + len(m.users)) % len(m.users)
}

func (m Model) openCreateUser() (tea.Model, tea.Cmd) {
	m.screen, m.err, m.notice = createUserScreen, nil, ""
	m.resetCreateUserForm()
	m.resizeForm()
	return m, m.form.Init()
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
