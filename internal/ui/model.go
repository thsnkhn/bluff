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

// Updates may need to resolve release metadata and download an archive. Keep
// that work isolated from the normal startup timeout so a slow update cannot
// make credential restoration look like a keychain failure.
const updateTimeout = 2 * time.Minute

type screen int

const (
	bootScreen screen = iota
	homeScreen
	loginScreen
	inviteCodeScreen
	inviteAccountScreen
	aboutScreen
	appMenuScreen
	usersScreen
	dashboardScreen
	tablesScreen
	tableDetailScreen
	formatsScreen
	playersScreen
	playerDetailScreen
	gamesScreen
	gameDetailScreen
	tableCreateScreen
	formatCreateScreen
	playerCreateScreen
	recordGameScreen
)

const (
	homeSignIn = iota
	homeHaveInvite
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

type inviteValues struct {
	code     string
	username string
	password string
}

type tableFormValues struct {
	name string
}

type formatFormValues struct {
	name          string
	requiredEntry string
	chips         []chipFormValue
}

type playerFormValues struct {
	name           string
	generateInvite bool
}

type chipFormValue struct {
	color string
	value string
}

type recordPhase int

const (
	recordDetailsPhase recordPhase = iota
	recordFormatPhase
	recordPlayersPhase
	recordChipCountsPhase
	recordReviewPhase
)

// recordPopupKind identifies the small metadata editors that sit on top of
// the recorder. Keep this separate from recordPhase so closing a popup never
// changes the recorder's current step.
type recordPopupKind int

const (
	recordPopupNone recordPopupKind = iota
	recordMetadataPopup
)

type recordDetailsValues struct {
	date    string
	remarks string
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
	HealthStatus(context.Context) (api.HealthStatus, error)
	Login(context.Context, string, string) (api.Session, error)
	ValidateInvitation(context.Context, string) error
	RedeemInvitation(context.Context, string, string, string) (api.Session, error)
	Me(context.Context, string) (api.User, error)
	Bootstrap(context.Context, string) (api.Bootstrap, error)
	Users(context.Context, string) ([]api.User, error)
	CreateInvitation(context.Context, string) (api.Invitation, error)
	Tables(context.Context, string) ([]api.TableSummary, error)
	Table(context.Context, string, string) (api.TableDetail, error)
	CreateTable(context.Context, string, string) (api.TableSummary, error)
	CreateTablePlayer(context.Context, string, string, string) (api.TablePlayer, error)
	UpdateTablePlayer(context.Context, string, string, string, string) (api.TablePlayer, error)
	DeleteTablePlayer(context.Context, string, string, string) error
	DisableTablePlayer(context.Context, string, string, string) error
	CreateGameFormat(context.Context, string, string, string, int, []api.ChipDenomination) (api.GameFormat, error)
	UpdateGameFormat(context.Context, string, string, string, string, int, []api.ChipDenomination) (api.GameFormat, error)
	PreviewTableGame(context.Context, string, string, string, string, string, []api.GameParticipantInput) (api.TableGame, error)
	RecordTableGame(context.Context, string, string, string, string, string, []api.GameParticipantInput) (api.TableDetail, error)
	PreviewTableGameEdit(context.Context, string, string, string, string, string, string, []api.GameParticipantInput) (api.TableGame, error)
	UpdateTableGame(context.Context, string, string, string, string, string, string, int, []api.GameParticipantInput) (api.TableDetail, error)
	Logout(context.Context, string) error
}

// UpdateInstaller installs a verified release and starts the replacement
// process. Keeping it behind an interface makes the boot flow cheap to test.
type UpdateInstaller interface {
	Install(context.Context, api.ClientRelease) error
}

// CredentialStore persists the bearer token outside the application files.
type CredentialStore interface {
	Load(context.Context) (string, error)
	Save(context.Context, string) error
	Delete(context.Context) error
}

// Model is the root Bubble Tea model.
type Model struct {
	api                 API
	store               CredentialStore
	build               BuildInfo
	updater             UpdateInstaller
	screen              screen
	width               int
	height              int
	spinner             spinner.Model
	form                *huh.Form
	login               *loginValues
	invite              *inviteValues
	loading             bool
	status              string
	err                 error
	token               string
	user                api.User
	bootstrap           api.Bootstrap
	homeIndex           int
	appIndex            int
	usersIndex          int
	usersActionHover    string
	users               []api.User
	tables              []api.TableSummary
	table               *api.TableDetail
	tableIndex          int
	tableNavIndex       int
	formatIndex         int
	formatEditIndex     int
	playerIndex         int
	gameIndex           int
	resultIndex         int
	tablesActionHover   string
	formatActionHover   string
	playerActionHover   string
	tableForm           *tableFormValues
	formatForm          *formatFormValues
	playerForm          *playerFormValues
	recordDetails       *recordDetailsValues
	recordMetadataDraft *recordDetailsValues
	recordPhase         recordPhase
	recordPopup         recordPopupKind
	recordBaseline      string
	recordFormatIndex   int
	recordPlayerIndex   int
	recordCounts        map[string]map[string]int
	recordEntered       map[string]bool
	recordAllIn         bool
	recordAllInValue    *bool
	recordClearConfirm  bool
	recordChipValues    []string
	recordPreview       *api.TableGame
	recordEditGameID    string
	recordEditVersion   int
	recordQuickAdd      bool
	recordQuickAddID    string
	playerInviteCodes   map[string]string
	playerInvitePopup   bool
	playerDeleteConfirm bool
	searchActive        bool
	searchQuery         string
	notice              string
	pendingTableNotice  string
	connected           bool
	checkingConnection  bool
	updateAvailable     *api.ClientRelease
}

// New constructs the Bluff terminal application.
func New(client API, store CredentialStore, build BuildInfo, installers ...UpdateInstaller) Model {
	busy := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	busy.Style = lipgloss.NewStyle().Foreground(colorFuchsia)
	model := Model{
		api:               client,
		store:             store,
		build:             build,
		screen:            bootScreen,
		spinner:           busy,
		loading:           true,
		status:            "Connecting",
		formatEditIndex:   -1,
		playerInviteCodes: make(map[string]string),
	}
	if len(installers) > 0 {
		model.updater = installers[0]
	}
	model.resetLoginForm()
	return model
}

// Init restores any saved session while starting the loading animation.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, m.spinner.Tick, m.bootHealthCmd())
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
		if m.searchActive {
			switch key {
			case "esc":
				m.searchActive, m.searchQuery = false, ""
				return m, nil
			case "enter":
				m.searchActive = false
				return m, nil
			case "backspace":
				m.searchQuery = trimLastRune(m.searchQuery)
				return m, nil
			default:
				if msg.Text != "" {
					m.searchQuery += msg.Text
					return m, nil
				}
			}
		}
		if key == "/" && m.isSearchableScreen() {
			m.searchActive, m.searchQuery = true, ""
			return m, nil
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
		if (m.screen == loginScreen || m.screen == inviteCodeScreen) && !m.loading && key == "esc" {
			previous := m.screen
			m.screen, m.err = homeScreen, nil
			if previous == loginScreen {
				m.resetLoginForm()
			} else {
				m.resetInviteCodeForm()
			}
			m.resizeForm()
			return m, nil
		}
		if m.screen == inviteAccountScreen && !m.loading && key == "esc" {
			m.screen, m.err = inviteCodeScreen, nil
			m.resetInviteCodeForm()
			m.resizeForm()
			return m, m.form.Init()
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
				m.usersActionHover = ""
				moveVisible(&m.usersIndex, m.visibleUserIndices(), -1)
				return m, nil
			case "down", "j":
				m.usersActionHover = ""
				moveVisible(&m.usersIndex, m.visibleUserIndices(), 1)
				return m, nil
			case "c":
				m.usersActionHover = ""
				m.loading, m.status, m.err, m.notice = true, "Creating invite code", nil, ""
				return m, tea.Batch(m.spinner.Tick, m.createInvitationCmd())
			case "r":
				m.usersActionHover = ""
				m.loading, m.status, m.err, m.notice = true, "Refreshing users", nil, ""
				return m, tea.Batch(m.spinner.Tick, m.loadUsersCmd())
			case "esc", "backspace":
				m.usersActionHover = ""
				if m.notice != "" {
					m.notice = ""
					return m, nil
				}
				m.screen, m.err, m.notice = appMenuScreen, nil, ""
				return m, nil
			}
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
		if !m.loading && m.isTableInteractiveScreen() {
			if updated, cmd, handled := m.updateTableKey(key); handled {
				return updated, cmd
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
			m.usersActionHover = ""
			m.usersIndex = msg.userIndex
			return m, nil
		}
		m.usersActionHover = msg.action
		if !msg.activate {
			return m, nil
		}
		m.usersActionHover = ""
		switch msg.action {
		case "search":
			m.searchActive, m.searchQuery = true, ""
			return m, nil
		case "create":
			m.loading, m.status, m.err, m.notice = true, "Creating invite code", nil, ""
			return m, tea.Batch(m.spinner.Tick, m.createInvitationCmd())
		case "refresh":
			m.loading, m.status, m.err, m.notice = true, "Refreshing users", nil, ""
			return m, tea.Batch(m.spinner.Tick, m.loadUsersCmd())
		case "back":
			m.screen, m.err, m.notice = appMenuScreen, nil, ""
			return m, nil
		}
		return m, nil
	case tableScrollMsg:
		if !m.loading && m.isTableListScreen() {
			m.scrollTableSelection(msg.delta)
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
	case tableMouseMsg:
		return m.updateTableMouse(msg)
	case startupHealthMsg:
		m.connected = msg.err == nil
		update := m.newerRelease(msg.health.ClientVersion)
		m.updateAvailable = update
		if update != nil && m.updater != nil {
			m.loading, m.status, m.err = true, "Downloading update "+update.Version, nil
			return m, tea.Batch(m.spinner.Tick, m.bootUpdateCmd(update, msg.err))
		}
		m.loading, m.status, m.err = true, "Restoring session", nil
		return m, tea.Batch(m.spinner.Tick, m.restoreSessionAfterHealthCmd(msg.err, update))
	case startupUpdateFailedMsg:
		// Keep the release indicator visible, but let a failed update fall back
		// to the normal startup path. The update error must not block login.
		m.loading, m.status, m.err = true, "Restoring session", nil
		return m, tea.Batch(m.spinner.Tick, m.restoreSessionAfterHealthCmd(msg.connectionErr, msg.update))
	case sessionRestoredMsg:
		m.loading = false
		m.updateAvailable = msg.update
		m.token, m.user, m.bootstrap = msg.token, msg.user, msg.bootstrap
		m.screen, m.err, m.connected = appMenuScreen, nil, true
		m.tables = msg.bootstrap.Tables
		m.searchActive, m.searchQuery = false, ""
		m.appIndex = m.firstEnabledAppIndex()
		return m, nil
	case loginRequiredMsg:
		m.loading, m.screen, m.err = false, homeScreen, msg.err
		m.connected = msg.connected
		m.updateAvailable = msg.update
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
		m.tables = msg.bootstrap.Tables
		m.searchActive, m.searchQuery = false, ""
		m.appIndex = m.firstEnabledAppIndex()
		m.login.password = ""
		if m.invite != nil {
			m.invite.password = ""
		}
		return m, nil
	case updateRestartedMsg:
		return m, tea.Quit
	case invitationValidatedMsg:
		m.loading, m.screen, m.err = false, inviteAccountScreen, nil
		m.resetInviteAccountForm()
		m.resizeForm()
		return m, m.form.Init()
	case operationFailedMsg:
		m.loading, m.err = false, msg.err
		if m.screen == loginScreen {
			m.login.password = ""
			m.resetLoginForm()
			m.resizeForm()
			return m, m.form.Init()
		}
		if m.screen == inviteCodeScreen {
			m.resetInviteCodeForm()
			m.resizeForm()
			return m, m.form.Init()
		}
		if m.screen == inviteAccountScreen {
			m.invite.password = ""
			m.resetInviteAccountForm()
			m.resizeForm()
			return m, m.form.Init()
		}
		if m.screen == playerDetailScreen && m.playerCanEdit() {
			m.resetPlayerEditForm()
			if m.form != nil {
				return m, m.form.Init()
			}
		}
		if m.isTableScreen() {
			m.loading = false
			return m, nil
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
	case invitationCreatedMsg:
		m.loading, m.err = false, nil
		m.screen, m.notice = usersScreen, "Invite code  "+msg.invitation.Code+"  ·  share it once"
		return m, nil
	case tablesLoadedMsg:
		m.loading, m.tables, m.err = false, msg.tables, nil
		if m.tableIndex >= len(m.tables) {
			m.tableIndex = max(len(m.tables)-1, 0)
		}
		return m, nil
	case tableLoadedMsg:
		if m.screen == tablesScreen {
			// Opening an existing table starts from the list screen. Switch views
			// only after the read model arrives so the list never flashes back over
			// a successfully loaded detail page.
			m.screen = tableDetailScreen
		}
		m.tableNavIndex = int(tableOverviewSection)
		m.loading, m.table, m.err, m.notice = false, &msg.table, nil, m.pendingTableNotice
		m.pendingTableNotice = ""
		if m.recordQuickAdd && m.recordQuickAddID != "" {
			for index, player := range m.table.Players {
				if player.ID == m.recordQuickAddID {
					m.playerIndex = index
					break
				}
			}
			m.recordQuickAdd, m.recordQuickAddID = false, ""
			m.screen, m.recordPhase = recordGameScreen, recordPlayersPhase
		}
		return m, nil
	case tableCreatedMsg:
		m.tables = append(m.tables, msg.table)
		m.tableIndex = len(m.tables) - 1
		// Move off the completed form before loading the new table. Otherwise the
		// form remains in huh.StateCompleted and the next keypress submits it again,
		// producing a misleading duplicate-table error after a successful create.
		m.screen = tableDetailScreen
		m.loading, m.status, m.err = true, "Opening table", nil
		return m, tea.Batch(m.spinner.Tick, m.tableCmd(msg.table.ID))
	case tablePlayerCreatedMsg:
		if m.playerInviteCodes == nil {
			m.playerInviteCodes = make(map[string]string)
		}
		if m.table != nil {
			alreadyPresent := false
			for _, player := range m.table.Players {
				if player.ID == msg.player.ID {
					alreadyPresent = true
					break
				}
			}
			if !alreadyPresent {
				m.table.Players = append(m.table.Players, msg.player)
			}
		}
		if m.recordQuickAdd {
			m.recordQuickAddID = msg.player.ID
		}
		m.screen = playersScreen
		m.form = nil
		if msg.inviteCode != "" {
			m.playerInviteCodes[msg.player.ID] = msg.inviteCode
			m.pendingTableNotice = "Invite code  " + msg.inviteCode + "  ·  share it once"
		}
		m.loading, m.status, m.err = false, "", nil
		if m.recordQuickAdd {
			m.playerIndex = len(m.table.Players) - 1
			m.recordQuickAdd = false
			m.screen, m.recordPhase = recordGameScreen, recordPlayersPhase
		}
		return m, nil
	case tablePlayerUpdatedMsg:
		if m.playerInviteCodes == nil {
			m.playerInviteCodes = make(map[string]string)
		}
		if m.table != nil {
			for index, player := range m.table.Players {
				if player.ID == msg.player.ID {
					m.table.Players[index] = msg.player
					m.playerIndex = index
					break
				}
			}
		}
		m.form = nil
		m.loading, m.status, m.err = false, "", nil
		if msg.inviteCode != "" {
			m.playerInviteCodes[msg.player.ID] = msg.inviteCode
			m.playerInvitePopup = true
		} else {
			m.resetPlayerEditForm()
			if m.form != nil {
				return m, m.form.Init()
			}
		}
		return m, nil
	case playerInviteCreatedMsg:
		if m.playerInviteCodes == nil {
			m.playerInviteCodes = make(map[string]string)
		}
		m.playerInviteCodes[msg.playerID] = msg.code
		m.playerInvitePopup = true
		m.loading, m.status, m.err = false, "", nil
		return m, nil
	case tablePlayerRemovedMsg:
		if m.table != nil {
			players := m.table.Players[:0]
			for _, player := range m.table.Players {
				if player.ID != msg.playerID {
					players = append(players, player)
				}
			}
			m.table.Players = players
			m.playerIndex = max(min(m.playerIndex, len(m.table.Players)-1), 0)
		}
		m.screen, m.loading, m.status, m.err = playersScreen, false, "", nil
		if msg.disabled {
			m.notice = "Player disabled"
		} else {
			m.notice = "Player deleted"
		}
		return m, nil
	case tableFormatCreatedMsg:
		if m.table != nil {
			m.table.Formats = append(m.table.Formats, msg.format)
		}
		m.formatEditIndex = -1
		m.screen = formatsScreen
		m.form = nil
		m.loading, m.status, m.err = false, "", nil
		return m, nil
	case tableFormatUpdatedMsg:
		if m.table != nil && msg.index >= 0 && msg.index < len(m.table.Formats) {
			m.table.Formats[msg.index] = msg.format
			m.formatIndex = msg.index
		}
		m.formatEditIndex = -1
		m.screen = formatsScreen
		m.form = nil
		m.loading, m.status, m.err = false, "", nil
		return m, nil
	case tableGamePreviewedMsg:
		m.loading, m.recordPhase, m.recordPreview, m.err = false, recordReviewPhase, &msg.game, nil
		m.resultIndex = 0
		return m, nil
	case tableGameRecordedMsg:
		m.loading, m.table, m.recordPreview, m.err, m.notice = false, &msg.table, nil, nil, "Game recorded"
		m.searchActive, m.searchQuery = false, ""
		m.screen = tableDetailScreen
		return m, nil
	case loggedOutMsg:
		m.loading, m.token, m.user, m.bootstrap = false, "", api.User{}, api.Bootstrap{}
		m.screen, m.err = homeScreen, msg.err
		m.tables, m.table = nil, nil
		m.searchActive, m.searchQuery = false, ""
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

	if (m.screen == loginScreen || m.screen == inviteCodeScreen || m.screen == inviteAccountScreen) && !m.loading {
		formMsg := msg
		if m.screen == inviteCodeScreen {
			switch typed := msg.(type) {
			case tea.KeyPressMsg:
				typed.Text = strings.ToUpper(typed.Text)
				formMsg = typed
			case tea.PasteMsg:
				typed.Content = strings.ToUpper(typed.Content)
				formMsg = typed
			}
		}
		updated, cmd := m.form.Update(formMsg)
		if form, ok := updated.(*huh.Form); ok {
			m.form = form
		}
		if m.screen == inviteCodeScreen {
			normalized := strings.ToUpper(m.invite.code)
			if normalized != m.invite.code {
				m.invite.code = normalized
				m.resetInviteCodeForm()
				m.resizeForm()
				return m, m.form.Init()
			}
		}
		if m.form.State == huh.StateCompleted {
			if m.screen == inviteCodeScreen {
				m.loading, m.status, m.err = true, "Checking invite code", nil
				return m, tea.Batch(m.spinner.Tick, m.validateInvitationCmd())
			}
			if m.screen == inviteAccountScreen {
				m.loading, m.status, m.err = true, "Creating your account", nil
				return m, tea.Batch(m.spinner.Tick, m.redeemInvitationCmd())
			}
			m.loading, m.status, m.err = true, "Taking your seat", nil
			return m, tea.Batch(m.spinner.Tick, m.loginCmd(m.login.username, m.login.password))
		}
		if m.form.State == huh.StateAborted {
			if m.screen == inviteAccountScreen {
				m.screen = inviteCodeScreen
			} else {
				m.screen = homeScreen
			}
			return m, nil
		}
		return m, cmd
	}

	if m.isTableFormScreen() && !m.loading && m.form != nil {
		allInBefore := m.recordAllIn
		updated, cmd := m.form.Update(msg)
		if form, ok := updated.(*huh.Form); ok {
			m.form = form
		}
		if m.screen == recordGameScreen && m.recordPhase == recordChipCountsPhase && m.recordAllInValue != nil {
			// Huh retains a pointer to field values while Bubble Tea copies Model
			// values between updates. Keep this boolean behind a stable pointer,
			// then synchronize it before deciding whether to rebuild the form.
			m.recordAllIn = *m.recordAllInValue
		}
		if m.screen == recordGameScreen && m.recordPhase == recordChipCountsPhase && m.recordAllIn != allInBefore {
			// The all-in choice hides or restores the denomination field. Rebuild
			// the compact form so the popup follows that choice immediately.
			m.rebuildRecordChipForm()
			return m, m.form.Init()
		}
		if (m.screen == playerCreateScreen || m.screen == playerDetailScreen) && m.playerForm != nil {
			m.playerForm.name = strings.ToLower(m.playerForm.name)
		}
		if m.form.State == huh.StateCompleted {
			return m.handleTableFormCompleted()
		}
		if m.form.State == huh.StateAborted {
			m.screen = m.tableParentScreen()
			m.err = nil
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
	case inviteCodeScreen:
		content = m.inviteCodeView()
	case inviteAccountScreen:
		content = m.inviteAccountView()
	case aboutScreen:
		content = m.aboutView()
	case appMenuScreen:
		content = m.appMenuView()
	case usersScreen:
		content = m.usersView()
	case dashboardScreen:
		content = m.dashboardView()
	case tablesScreen:
		content = m.tablesView()
	case tableDetailScreen:
		content = m.tableDetailView()
	case formatsScreen:
		content = m.formatsView()
	case playersScreen:
		content = m.playersView()
	case gamesScreen:
		content = m.gamesView()
	case gameDetailScreen:
		content = m.gameDetailView()
	case tableCreateScreen:
		content = m.tableCreateView()
	case formatCreateScreen:
		content = m.formatCreateView()
	case playerCreateScreen:
		content = m.playerCreateView()
	case playerDetailScreen:
		content = m.playerDetailView()
	case recordGameScreen:
		content = m.recordGameView()
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
	m.form = newHuhForm(huh.NewGroup(
		newHuhInput("Username", "", "bluff", &m.login.username, 32, false,
			required("enter your username")),
		newHuhInput("Password", "", "••••••••", &m.login.password, 128, true,
			required("enter your password")),
	))
}

func (m *Model) resetInviteCodeForm() {
	code := ""
	if m.invite != nil {
		code = m.invite.code
	}
	m.invite = &inviteValues{code: code}
	m.form = newHuhForm(huh.NewGroup(
		newHuhInput("Invite code", "", "A1B2C3", &m.invite.code, 6, false,
			inviteCode),
	))
}

func (m *Model) resetInviteAccountForm() {
	code, username := m.invite.code, m.invite.username
	m.invite = &inviteValues{code: code, username: username}
	m.form = newHuhForm(huh.NewGroup(
		newHuhInput("Username", "", "table-friend",
			&m.invite.username, 32, false, required("enter a username")),
		newHuhInput("Password", "", "••••••••", &m.invite.password, 128, true, validPassword),
	))
}

func centeredFormTheme(isDark bool) *huh.Styles {
	styles := huh.ThemeCharm(isDark)
	center := func(field *huh.FieldStyles) {
		field.Base = field.Base.BorderLeft(false).BorderBottom(true).PaddingLeft(0).Align(lipgloss.Center)
	}
	center(&styles.Focused)
	center(&styles.Blurred)
	styles.Form.Base = styles.Form.Base.Align(lipgloss.Center)
	styles.Group.Base = styles.Group.Base.Align(lipgloss.Center)
	return styles
}

func popupConfirm(title string, value *bool) *huh.Confirm {
	return huh.NewConfirm().
		Title(title).
		Affirmative("Yes").
		Negative("No").
		Value(value).
		// Keep the choice centered inside the field and remove the empty row
		// between its title and the buttons.
		Inline(true).
		WithButtonAlignment(lipgloss.Center)
}

func popupStackedConfirm(title string, value *bool) *huh.Confirm {
	return huh.NewConfirm().
		// Huh's default stacked confirm inserts a blank row between the title
		// and buttons. A trailing newline with inline rendering keeps the
		// options directly below the label.
		Title(title + "\n").
		Affirmative("Yes").
		Negative("No").
		Value(value).
		Inline(true)
}

func (m *Model) resizeForm() {
	if m.form == nil {
		return
	}
	width := min(max(m.width-16, 32), 54)
	// Recorder metadata uses compact overlay forms. Giving these forms the
	// recorder page height makes a one-line date or note field fill almost the
	// entire terminal.
	if m.screen == recordGameScreen && m.recordPopup != recordPopupNone {
		m.form.WithWidth(width)
		return
	}
	// The earnings counter is a single custom field. Let Huh keep its natural
	// height so the popup grows with the number of denominations instead of
	// inheriting the tall record form viewport.
	if m.screen == recordGameScreen && m.recordPhase == recordChipCountsPhase {
		m.form.WithWidth(width)
		return
	}
	// Player editors are rendered inside compact popups. Let Huh use the
	// group's natural height instead of reserving the full page viewport.
	if m.screen == playerCreateScreen || m.screen == playerDetailScreen {
		m.form.WithWidth(width)
		return
	}
	height := 16
	if m.screen == inviteAccountScreen {
		height = 18
	}
	if m.screen == formatCreateScreen {
		height = 14
	}
	if m.screen == recordGameScreen {
		height = 34
	}
	m.form.WithWidth(width).WithHeight(min(max(m.height-18, 8), height))
}

func inviteCode(value string) error {
	if len(value) != 6 {
		return errors.New("enter all 6 characters")
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') {
			continue
		}
		return errors.New("use letters and numbers only")
	}
	return nil
}

func validPassword(value string) error {
	if len(value) < 8 {
		return errors.New("use at least 8 characters")
	}
	if len(value) > 128 {
		return errors.New("use no more than 128 characters")
	}
	return nil
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
	update    *api.ClientRelease
}

type loginRequiredMsg struct {
	err       error
	connected bool
	update    *api.ClientRelease
}
type startupHealthMsg struct {
	health api.HealthStatus
	err    error
}
type startupUpdateFailedMsg struct {
	update        *api.ClientRelease
	connectionErr error
	err           error
}
type connectionCheckedMsg struct{ err error }
type updateRestartedMsg struct{}
type loginSucceededMsg sessionRestoredMsg
type invitationValidatedMsg struct{}
type operationFailedMsg struct{ err error }
type refreshedMsg struct{ bootstrap api.Bootstrap }
type usersLoadedMsg struct{ users []api.User }
type invitationCreatedMsg struct{ invitation api.Invitation }
type loggedOutMsg struct{ err error }

func (m Model) bootHealthCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		health, err := m.api.HealthStatus(ctx)
		return startupHealthMsg{health: health, err: err}
	}
}

func (m Model) bootUpdateCmd(update *api.ClientRelease, connectionErr error) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
		defer cancel()
		if err := m.updater.Install(ctx, *update); err != nil {
			return startupUpdateFailedMsg{update: update, connectionErr: connectionErr, err: err}
		}
		return updateRestartedMsg{}
	}
}

func (m Model) restoreSessionAfterHealthCmd(connectionErr error, update *api.ClientRelease) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		return m.restoreSessionMessage(ctx, connectionErr, update)
	}
}

func (m Model) restoreSessionMessage(ctx context.Context, connectionErr error, update *api.ClientRelease) tea.Msg {
	token, err := m.store.Load(ctx)
	if errors.Is(err, credentials.ErrNotFound) {
		return loginRequiredMsg{err: connectionErr, connected: connectionErr == nil, update: update}
	}
	if err != nil {
		return loginRequiredMsg{err: fmt.Errorf("could not open the system keychain: %w", err), connected: connectionErr == nil, update: update}
	}
	user, err := m.api.Me(ctx, token)
	if err != nil {
		if api.IsUnauthorized(err) {
			_ = m.store.Delete(ctx)
			return loginRequiredMsg{err: errors.New("your saved session expired; sign in again"), connected: connectionErr == nil, update: update}
		}
		return loginRequiredMsg{err: err, connected: connectionErr == nil, update: update}
	}
	bootstrap, err := m.api.Bootstrap(ctx, token)
	if err != nil {
		return loginRequiredMsg{err: err, connected: connectionErr == nil, update: update}
	}
	return sessionRestoredMsg{token: token, user: user, bootstrap: bootstrap, update: update}
}

func (m Model) restoreSessionCmd() tea.Cmd {
	return func() tea.Msg {
		healthCtx, healthCancel := context.WithTimeout(context.Background(), operationTimeout)
		health, connectionErr := m.api.HealthStatus(healthCtx)
		healthCancel()
		update := m.newerRelease(health.ClientVersion)
		if update != nil && m.updater != nil {
			updateCtx, updateCancel := context.WithTimeout(context.Background(), updateTimeout)
			installErr := m.updater.Install(updateCtx, *update)
			updateCancel()
			if installErr == nil {
				return updateRestartedMsg{}
			}
		}

		sessionCtx, sessionCancel := context.WithTimeout(context.Background(), operationTimeout)
		defer sessionCancel()
		return m.restoreSessionMessage(sessionCtx, connectionErr, update)
	}
}

func (m Model) checkConnectionCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		_, err := m.api.HealthStatus(ctx)
		return connectionCheckedMsg{err: err}
	}
}

func (m Model) activateHomeItem() (tea.Model, tea.Cmd) {
	switch m.homeIndex {
	case homeSignIn:
		m.screen, m.err = loginScreen, nil
		m.resetLoginForm()
		m.resizeForm()
		return m, m.form.Init()
	case homeHaveInvite:
		m.screen, m.err = inviteCodeScreen, nil
		m.resetInviteCodeForm()
		m.resizeForm()
		return m, m.form.Init()
	case homeCheckConnection:
		m.loading, m.checkingConnection, m.status, m.err = true, true, "Connecting", nil
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

func (m Model) validateInvitationCmd() tea.Cmd {
	code := m.invite.code
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		if err := m.api.ValidateInvitation(ctx, code); err != nil {
			return operationFailedMsg{err: err}
		}
		return invitationValidatedMsg{}
	}
}

func (m Model) redeemInvitationCmd() tea.Cmd {
	code, username, password := m.invite.code, m.invite.username, m.invite.password
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		session, err := m.api.RedeemInvitation(ctx, code, username, password)
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

func (m Model) createInvitationCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		invitation, err := m.api.CreateInvitation(ctx, m.token)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return invitationCreatedMsg{invitation: invitation}
	}
}

func (m Model) appMenuItems() []appMenuItem {
	items := make([]appMenuItem, 0, 7)
	if m.user.Role == "admin" {
		items = append(items, appMenuItem{title: "Users", action: actionUsers, enabled: true})
	}
	items = append(items,
		appMenuItem{title: "Tables", action: actionTables, enabled: true},
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
	case actionTables:
		m.screen, m.loading, m.status, m.err, m.notice = tablesScreen, true, "Loading tables", nil, ""
		return m, tea.Batch(m.spinner.Tick, m.tablesCmd())
	case actionLogout:
		m.loading, m.status, m.err = true, "Closing your session", nil
		return m, tea.Batch(m.spinner.Tick, m.logoutCmd())
	case actionQuit:
		return m, tea.Quit
	default:
		return m, nil
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
