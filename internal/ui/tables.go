package ui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	bubblesTable "charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/thsnkhn/bluff/internal/api"
)

type tableMouseMsg struct {
	action   string
	index    int
	activate bool
}
type tableScrollMsg struct{ delta int }

type tablesLoadedMsg struct{ tables []api.TableSummary }
type tableLoadedMsg struct{ table api.TableDetail }
type tableCreatedMsg struct{ table api.TableSummary }
type tablePlayerCreatedMsg struct {
	player     api.TablePlayer
	inviteCode string
}
type tablePlayerUpdatedMsg struct {
	player     api.TablePlayer
	inviteCode string
}
type playerInviteCreatedMsg struct {
	playerID string
	code     string
}
type tablePlayerRemovedMsg struct {
	playerID string
	disabled bool
}
type tableFormatCreatedMsg struct{ format api.GameFormat }
type tableFormatUpdatedMsg struct {
	index  int
	format api.GameFormat
}
type tableGamePreviewedMsg struct{ game api.TableGame }
type tableGameRecordedMsg struct{ table api.TableDetail }

func (m Model) isTableScreen() bool {
	switch m.screen {
	case tablesScreen, tableDetailScreen, formatsScreen, playersScreen,
		gamesScreen, gameDetailScreen, tableCreateScreen, formatCreateScreen, playerCreateScreen, playerDetailScreen, recordGameScreen:
		return true
	default:
		return false
	}
}

func (m Model) isTableInteractiveScreen() bool { return m.isTableScreen() && !m.loading }

func (m Model) isTableListScreen() bool {
	switch m.screen {
	case usersScreen, tablesScreen, formatsScreen, playersScreen, gamesScreen:
		return true
	default:
		return false
	}
}

func (m *Model) scrollTableSelection(delta int) {
	switch m.screen {
	case usersScreen:
		moveVisible(&m.usersIndex, m.visibleUserIndices(), delta)
	case tablesScreen:
		moveVisible(&m.tableIndex, m.visibleTableIndices(), delta)
	case formatsScreen:
		moveVisible(&m.formatIndex, m.visibleFormatIndices(), delta)
	case playersScreen:
		moveVisible(&m.playerIndex, m.visiblePlayerIndices(), delta)
	case gamesScreen:
		// Games render newest-first, opposite their source slice order.
		moveVisible(&m.gameIndex, m.visibleGameIndices(), -delta)
	}
}

func (m Model) tableParentScreen() screen {
	if m.screen == playerCreateScreen && m.recordQuickAdd {
		return recordGameScreen
	}
	switch m.screen {
	case formatCreateScreen:
		return formatsScreen
	case playerCreateScreen, playerDetailScreen:
		return playersScreen
	case formatsScreen, playersScreen, gamesScreen, gameDetailScreen:
		return tableDetailScreen
	case recordGameScreen:
		return tableDetailScreen
	default:
		return tablesScreen
	}
}

func (m Model) updateTableKey(key string) (tea.Model, tea.Cmd, bool) {
	if m.screen == tableDetailScreen && m.expandedChart != tableChartNone {
		switch key {
		case "esc", "backspace":
			m.expandedChart = tableChartNone
			return m, nil, true
		case "left", "right":
			if m.expandedChart == tableChartStandings {
				if key == "left" {
					m.standingEntryOffset = min(m.standingEntryOffset+1, max(len(m.table.Games)-1, 0))
				} else {
					m.standingEntryOffset = max(m.standingEntryOffset-1, 0)
				}
			}
			return m, nil, true
		case "alt+h":
			m.expandedChart = tableChartHistory
			return m, nil, true
		case "alt+s":
			m.expandedChart = tableChartStandings
			m.standingEntryOffset = 0
			return m, nil, true
		default:
			// Focus mode owns the overview until it is closed. In particular,
			// tab must not navigate away while a chart is expanded.
			return m, nil, true
		}
	}
	if (m.screen == tableDetailScreen || m.screen == formatsScreen || m.screen == playersScreen || m.screen == gamesScreen) && (key == "tab" || key == "shift+tab") {
		direction := 1
		if key == "shift+tab" {
			direction = -1
		}
		updated, cmd := m.cycleTableSection(direction)
		return updated, cmd, true
	}
	switch m.screen {
	case tableCreateScreen, formatCreateScreen, playerCreateScreen:
		if key == "esc" {
			m.screen, m.err = m.tableParentScreen(), nil
			return m, nil, true
		}
		return m, nil, false
	case tablesScreen:
		switch key {
		case "up", "k":
			moveVisible(&m.tableIndex, m.visibleTableIndices(), -1)
			return m, nil, true
		case "down", "j":
			moveVisible(&m.tableIndex, m.visibleTableIndices(), 1)
			return m, nil, true
		case "enter", " ":
			visible := m.visibleTableIndices()
			if len(visible) == 0 {
				return m, nil, true
			}
			m.loading, m.status, m.err = true, "Opening table", nil
			return m, tea.Batch(m.spinner.Tick, m.tableCmd(m.tables[m.tableIndex].ID)), true
		case "c":
			m.screen, m.err = tableCreateScreen, nil
			m.resetTableCreateForm()
			return m, m.form.Init(), true
		case "r":
			m.loading, m.status, m.err = true, "Refreshing tables", nil
			return m, tea.Batch(m.spinner.Tick, m.tablesCmd()), true
		case "esc", "backspace":
			m.screen, m.err = appMenuScreen, nil
			return m, nil, true
		}
	case tableDetailScreen:
		if m.table == nil {
			return m, nil, true
		}
		switch key {
		case "alt+h":
			if len(m.table.Games) > 0 {
				m.expandedChart = tableChartHistory
			}
			return m, nil, true
		case "alt+s":
			if len(m.table.Games) > 0 {
				m.expandedChart = tableChartStandings
				m.standingEntryOffset = 0
			}
			return m, nil, true
		case "r":
			m.loading, m.status, m.err = true, "Refreshing table", nil
			return m, tea.Batch(m.spinner.Tick, m.tableCmd(m.table.Table.ID)), true
		case "c":
			if m.table.CanManage {
				m.startRecordGame()
				return m, nil, true
			}
		case "esc", "backspace":
			m.screen, m.err, m.expandedChart = tablesScreen, nil, tableChartNone
			return m, nil, true
		}
	case formatsScreen:
		switch key {
		case "up", "k":
			moveVisible(&m.formatIndex, m.visibleFormatIndices(), -1)
			return m, nil, true
		case "down", "j":
			moveVisible(&m.formatIndex, m.visibleFormatIndices(), 1)
			return m, nil, true
		case "enter", " ":
			if len(m.table.Formats) > 0 && m.table.CanManage {
				m.screen = formatCreateScreen
				m.resetFormatEditForm()
				return m, m.form.Init(), true
			}
			return m, nil, true
		case "c":
			if m.table.CanManage {
				m.screen, m.err = formatCreateScreen, nil
				m.resetFormatCreateForm()
				return m, m.form.Init(), true
			}
		case "esc", "backspace":
			m.screen, m.err = tableDetailScreen, nil
			return m, nil, true
		}
	case gamesScreen:
		switch key {
		case "up", "k":
			moveVisible(&m.gameIndex, m.visibleGameIndices(), 1)
			return m, nil, true
		case "down", "j":
			moveVisible(&m.gameIndex, m.visibleGameIndices(), -1)
			return m, nil, true
		case "enter", " ":
			if len(m.visibleGameIndices()) > 0 {
				m.screen, m.resultIndex = gameDetailScreen, 0
			}
			return m, nil, true
		case "c":
			if m.table.CanManage {
				m.startRecordGame()
				return m, nil, true
			}
		case "esc", "backspace":
			m.screen, m.err = tableDetailScreen, nil
			return m, nil, true
		}
	case gameDetailScreen:
		switch key {
		case "up", "k":
			m.moveResultSelection(-1, m.inspectionPlayerCount())
			return m, nil, true
		case "down", "j":
			m.moveResultSelection(1, m.inspectionPlayerCount())
			return m, nil, true
		case "e":
			if m.canEditSelectedGame() {
				m.startEditSelectedGame()
				return m, nil, true
			}
		case "esc", "backspace":
			m.screen, m.err = gamesScreen, nil
			return m, nil, true
		}
	case playersScreen:
		switch key {
		case "up", "k":
			moveVisible(&m.playerIndex, m.visiblePlayerIndices(), -1)
			return m, nil, true
		case "down", "j":
			moveVisible(&m.playerIndex, m.visiblePlayerIndices(), 1)
			return m, nil, true
		case "enter", " ":
			if len(m.visiblePlayerIndices()) > 0 {
				m.screen = playerDetailScreen
				m.resetPlayerEditForm()
				if m.form != nil {
					return m, m.form.Init(), true
				}
			}
			return m, nil, true
		case "c":
			if m.table.CanManage {
				m.screen, m.err = playerCreateScreen, nil
				m.resetPlayerCreateForm()
				return m, m.form.Init(), true
			}
		case "esc", "backspace":
			m.screen, m.err = tableDetailScreen, nil
			return m, nil, true
		}
	case playerDetailScreen:
		if m.playerInvitePopup {
			if key == "esc" || key == "backspace" || key == "enter" {
				m.playerInvitePopup = false
				m.resetPlayerEditForm()
				if m.form != nil {
					return m, m.form.Init(), true
				}
				return m, nil, true
			}
			return m, nil, true
		}
		if m.playerDeleteConfirm && key != "alt+d" {
			m.playerDeleteConfirm = false
		}
		if key == "enter" && m.form != nil && m.table.CanManage && m.playerCanEdit() {
			updated, cmd := m.handleTableFormCompleted()
			return updated, cmd, true
		}
		if key == "c" && m.table.CanManage && m.playerInviteCodeForCurrentPlayer() == "" {
			m.loading, m.status, m.err = true, "Creating invite code", nil
			return m, tea.Batch(m.spinner.Tick, m.createPlayerInviteCmd()), true
		}
		if key == "alt+d" && m.table.CanManage && !m.playerHasEntries() {
			if !m.playerDeleteConfirm {
				m.playerDeleteConfirm = true
				return m, nil, true
			}
			m.playerDeleteConfirm = false
			m.loading, m.status, m.err = true, "Deleting player", nil
			return m, tea.Batch(m.spinner.Tick, m.deletePlayerCmd()), true
		}
		if key == "x" && m.table.CanManage && m.playerHasEntries() {
			m.loading, m.status, m.err = true, "Disabling player", nil
			return m, tea.Batch(m.spinner.Tick, m.disablePlayerCmd()), true
		}
		if key == "esc" || key == "backspace" {
			m.screen, m.err = playersScreen, nil
			m.form = nil
			return m, nil, true
		}
	case recordGameScreen:
		// Metadata editors are overlays on the recorder. Escape only dismisses
		// the overlay; it must never route the user back to the table overview.
		if m.recordPopup != recordPopupNone {
			if key == "esc" || key == "backspace" {
				m.form, m.recordPopup, m.recordMetadataDraft, m.err = nil, recordPopupNone, nil, nil
				return m, nil, true
			}
			return m, nil, false
		}
		if key == "t" && m.recordPhase != recordChipCountsPhase && m.recordPhase != recordDetailsPhase && m.recordPhase != recordReviewPhase {
			return m, m.openRecordMetadataPopup(), true
		}
		switch m.recordPhase {
		case recordDetailsPhase:
			// Kept as a compatibility guard for older in-memory models. New
			// recorder sessions always start at format selection.
			m.form, m.recordPhase, m.err = nil, recordFormatPhase, nil
			return m, nil, true
		case recordChipCountsPhase:
			if m.recordClearConfirm && key != "alt+d" {
				m.recordClearConfirm = false
			}
			if key == "alt+d" && m.currentRecordPlayerIsEntered() {
				if !m.recordClearConfirm {
					m.recordClearConfirm = true
					return m, nil, true
				}
				m.clearCurrentRecordPlayerEarnings()
				return m, nil, true
			}
			if key == "esc" || key == "backspace" {
				// Close only the earnings popup. Keep the recorder and its
				// saved player entries in place so the user can continue the game.
				m.form, m.err = nil, nil
				m.recordPhase = recordPlayersPhase
				return m, nil, true
			}
			return m, nil, false
		case recordFormatPhase:
			switch key {
			case "up", "k":
				m.recordFormatIndex = max(m.recordFormatIndex-1, 0)
				return m, nil, true
			case "down", "j":
				m.recordFormatIndex = min(m.recordFormatIndex+1, max(len(m.table.Formats)-1, 0))
				return m, nil, true
			case "enter", " ":
				if len(m.table.Formats) == 0 {
					return m, nil, true
				}
				m.recordPhase, m.playerIndex = recordPlayersPhase, 0
				return m, nil, true
			case "esc", "backspace":
				m.screen, m.err = tableDetailScreen, nil
				return m, nil, true
			}
		case recordPlayersPhase:
			switch key {
			case "up", "k":
				moveVisible(&m.playerIndex, m.visiblePlayerIndices(), -1)
				return m, nil, true
			case "down", "j":
				moveVisible(&m.playerIndex, m.visiblePlayerIndices(), 1)
				return m, nil, true
			case "enter", "e":
				visible := m.visiblePlayerIndices()
				if len(visible) == 0 {
					return m, nil, true
				}
				if !slices.Contains(visible, m.playerIndex) {
					m.playerIndex = visible[0]
				}
				return m, m.beginRecordPlayerEarnings(), true
			case "r":
				if m.hasEnoughRecordings() && m.recordHasChanges() {
					m.loading, m.status, m.err = true, "Checking table balance", nil
					return m, tea.Batch(m.spinner.Tick, m.previewRecordGameCmd()), true
				}
				return m, nil, true
			case "c":
				if m.table.CanManage && m.recordEditGameID == "" {
					m.recordQuickAdd = true
					m.screen, m.err = playerCreateScreen, nil
					m.resetPlayerCreateForm()
					return m, m.form.Init(), true
				}
			case "esc", "backspace":
				m.screen, m.err = tableDetailScreen, nil
				return m, nil, true
			}
		case recordReviewPhase:
			switch key {
			case "up", "k":
				m.moveResultSelection(-1, len(m.recordedPlayerIndices()))
				return m, nil, true
			case "down", "j":
				m.moveResultSelection(1, len(m.recordedPlayerIndices()))
				return m, nil, true
			case "esc", "backspace":
				m.screen, m.err = tableDetailScreen, nil
				return m, nil, true
			case "e":
				m.recordPhase, m.recordPreview, m.err = recordPlayersPhase, nil, nil
				return m, nil, true
			case "enter", "s":
				m.loading, m.err = true, nil
				m.status = "Recording game"
				if m.recordEditGameID != "" {
					m.status = "Updating game"
				}
				return m, tea.Batch(m.spinner.Tick, m.recordGameCmd()), true
			}
			return m, nil, false
		}
	}
	return m, nil, false
}

func (m Model) cycleTableSection(direction int) (tea.Model, tea.Cmd) {
	section := int(m.currentTableSection()) + direction
	if section < int(tableOverviewSection) {
		section = int(tableFormatsSection)
	}
	if section > int(tableFormatsSection) {
		section = int(tableOverviewSection)
	}
	return m.navigateTableSection(tableSection(section))
}

func (m Model) navigateTableSection(section tableSection) (tea.Model, tea.Cmd) {
	m.tableNavIndex = int(section)
	m.expandedChart = tableChartNone
	switch section {
	case tablePlayersSection:
		m.screen, m.playerIndex, m.err = playersScreen, 0, nil
	case tableFormatsSection:
		m.screen, m.formatIndex, m.err = formatsScreen, 0, nil
	case tableGamesSection:
		m.screen, m.gameIndex, m.err = gamesScreen, max(len(m.table.Games)-1, 0), nil
	default:
		m.screen, m.err = tableDetailScreen, nil
	}
	m.searchActive, m.searchQuery = false, ""
	return m, nil
}

func (m Model) handleTableFormCompleted() (tea.Model, tea.Cmd) {
	switch m.screen {
	case tableCreateScreen:
		m.loading, m.status, m.err = true, "Creating table", nil
		return m, tea.Batch(m.spinner.Tick, m.createTableCmd())
	case formatCreateScreen:
		if chips, err := parseChipRows(m.formatForm.chips); err != nil {
			m.err = err
			if m.formatEditIndex >= 0 {
				m.resetFormatEditForm()
			} else {
				m.resetFormatCreateForm()
			}
			return m, m.form.Init()
		} else {
			_ = chips
		}
		if m.formatEditIndex >= 0 && m.formatEditIndex < len(m.table.Formats) {
			m.loading, m.status, m.err = true, "Saving game format", nil
			return m, tea.Batch(m.spinner.Tick, m.updateFormatCmd())
		}
		m.loading, m.status, m.err = true, "Creating game format", nil
		return m, tea.Batch(m.spinner.Tick, m.createFormatCmd())
	case playerCreateScreen:
		m.loading, m.status, m.err = true, "Adding player", nil
		return m, tea.Batch(m.spinner.Tick, m.createPlayerCmd())
	case playerDetailScreen:
		if !m.playerCanEdit() {
			return m, nil
		}
		m.loading, m.status, m.err = true, "Saving player", nil
		return m, tea.Batch(m.spinner.Tick, m.updatePlayerCmd())
	case recordGameScreen:
		if m.recordPopup != recordPopupNone {
			// The date/note editor uses a draft so closing it never mutates the
			// recorder. Save the draft and return to the phase that opened it.
			if m.recordMetadataDraft != nil {
				m.recordDetails = m.recordMetadataDraft
			}
			m.recordMetadataDraft = nil
			// Metadata editors are local forms. Save them and
			// return to the same recorder phase that opened the popup.
			m.form, m.recordPopup, m.err = nil, recordPopupNone, nil
			return m, nil
		}
		if m.recordPhase == recordDetailsPhase {
			m.form = nil
			m.recordPhase, m.err = recordFormatPhase, nil
			return m, nil
		}
		if m.recordPhase == recordChipCountsPhase {
			player := m.table.Players[m.recordPlayerIndex]
			format := m.table.Formats[m.recordFormatIndex]
			counts := map[string]int{}
			if !m.recordAllIn {
				for index, chip := range format.Chips {
					value, err := strconv.Atoi(strings.TrimSpace(m.recordChipValues[index]))
					if err != nil || value < 0 {
						m.err = errors.New("chip counts must be whole numbers")
						return m, nil
					}
					counts[chip.ID] = value
				}
			}
			m.recordCounts[player.ID] = counts
			m.recordEntered[player.ID] = true
			// Return to the player list after every save. This makes the same
			// popup usable for both new entries and edits.
			m.form, m.recordPreview, m.err = nil, nil, nil
			m.recordPhase = recordPlayersPhase
			return m, nil
		}
	}
	return m, nil
}

func (m Model) recordedPlayerIndices() []int {
	indices := make([]int, 0, len(m.recordEntered))
	for index, player := range m.table.Players {
		if m.recordEntered[player.ID] {
			indices = append(indices, index)
		}
	}
	return indices
}

func (m Model) recordPlayerIsEntered() bool {
	if m.playerIndex < 0 || m.playerIndex >= len(m.table.Players) {
		return false
	}
	player := m.table.Players[m.playerIndex]
	return m.recordEntered[player.ID]
}

func (m Model) currentRecordPlayerIsEntered() bool {
	if m.table == nil || m.recordPlayerIndex < 0 || m.recordPlayerIndex >= len(m.table.Players) {
		return false
	}
	return m.recordEntered[m.table.Players[m.recordPlayerIndex].ID]
}

func (m *Model) clearCurrentRecordPlayerEarnings() {
	if !m.currentRecordPlayerIsEntered() {
		return
	}
	player := m.table.Players[m.recordPlayerIndex]
	delete(m.recordCounts, player.ID)
	delete(m.recordEntered, player.ID)
	m.recordPreview, m.err = nil, nil
	m.recordClearConfirm = false
	m.form = nil
	m.recordPhase = recordPlayersPhase
}

// beginRecordPlayerEarnings opens the same counter popup for a new entry or
// an existing player's saved earnings. Keeping this transition in one place
// keeps keyboard and mouse activation consistent.
func (m *Model) beginRecordPlayerEarnings() tea.Cmd {
	if m.table == nil || m.playerIndex < 0 || m.playerIndex >= len(m.table.Players) {
		return nil
	}
	m.recordPlayerIndex = m.playerIndex
	m.recordPhase = recordChipCountsPhase
	m.recordClearConfirm = false
	m.resetRecordChipForm()
	if m.form == nil {
		return nil
	}
	return m.form.Init()
}

func (m Model) hasEnoughRecordings() bool {
	return len(m.recordedPlayerIndices()) >= 2
}

func (m Model) recordHasChanges() bool {
	return m.recordStateSignature() != m.recordBaseline
}

func (m Model) recordStateSignature() string {
	var signature strings.Builder
	if m.recordDetails != nil {
		fmt.Fprintf(&signature, "date=%q;note=%q;", m.recordDetails.date, m.recordDetails.remarks)
	}
	fmt.Fprintf(&signature, "format=%d;", m.recordFormatIndex)
	if m.table == nil {
		return signature.String()
	}
	var chips []api.ChipDenomination
	if m.recordFormatIndex >= 0 && m.recordFormatIndex < len(m.table.Formats) {
		chips = m.table.Formats[m.recordFormatIndex].Chips
	}
	for _, player := range m.table.Players {
		if !m.recordEntered[player.ID] {
			continue
		}
		fmt.Fprintf(&signature, "player=%q;", player.ID)
		for _, chip := range chips {
			fmt.Fprintf(&signature, "chip=%q:%d;", chip.ID, m.recordCounts[player.ID][chip.ID])
		}
	}
	return signature.String()
}

func (m *Model) startRecordGame() {
	m.screen, m.recordPhase, m.recordPreview, m.err = recordGameScreen, recordFormatPhase, nil, nil
	m.recordFormatIndex, m.recordPlayerIndex, m.playerIndex = 0, 0, 0
	m.recordCounts = map[string]map[string]int{}
	m.recordEntered = map[string]bool{}
	m.recordAllIn = false
	m.recordAllInValue = nil
	m.recordClearConfirm = false
	m.recordPopup = recordPopupNone
	m.recordEditGameID, m.recordEditVersion = "", 0
	m.recordQuickAdd, m.recordQuickAddID = false, ""
	// The recorder opens on format selection. Metadata stays on the recorder
	// and is edited through its date/note popups.
	m.recordDetails = &recordDetailsValues{date: time.Now().Format("2006-01-02")}
	m.form = nil
	m.recordBaseline = m.recordStateSignature()
}

func (m Model) canEditSelectedGame() bool {
	return m.table != nil && m.table.CanManage && m.gameIndex >= 0 &&
		m.gameIndex == len(m.table.Games)-1
}

func (m *Model) startEditSelectedGame() {
	if !m.canEditSelectedGame() {
		return
	}
	game := m.table.Games[m.gameIndex]
	m.screen, m.recordPhase, m.recordPreview, m.err = recordGameScreen, recordPlayersPhase, nil, nil
	m.recordEditGameID, m.recordEditVersion = game.ID, game.Version
	m.recordFormatIndex = 0
	for index, format := range m.table.Formats {
		if format.ID == game.Format.ID {
			m.recordFormatIndex = index
			break
		}
	}
	m.recordCounts = make(map[string]map[string]int, len(game.Participants))
	m.recordEntered = make(map[string]bool, len(game.Participants))
	for _, participant := range game.Participants {
		counts := map[string]int{}
		for _, chip := range participant.ChipCounts {
			if chip.Count > 0 {
				counts[chip.DenominationID] = chip.Count
			}
		}
		m.recordCounts[participant.PlayerID] = counts
		m.recordEntered[participant.PlayerID] = true
	}
	m.playerIndex, m.recordPlayerIndex = 0, 0
	m.recordAllIn, m.recordAllInValue, m.recordClearConfirm = false, nil, false
	m.recordPopup, m.recordQuickAdd, m.recordQuickAddID = recordPopupNone, false, ""
	m.recordDetails = &recordDetailsValues{date: game.Date, remarks: game.Remarks}
	m.form = nil
	m.recordBaseline = m.recordStateSignature()
}

func (m Model) updateTableMouse(msg tableMouseMsg) (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	if strings.HasPrefix(msg.action, "nav:") {
		if !msg.activate {
			return m, nil
		}
		switch strings.TrimPrefix(msg.action, "nav:") {
		case "players":
			return m.navigateTableSection(tablePlayersSection)
		case "formats":
			return m.navigateTableSection(tableFormatsSection)
		case "games":
			return m.navigateTableSection(tableGamesSection)
		case "overview":
			return m.navigateTableSection(tableOverviewSection)
		}
	}
	switch m.screen {
	case tablesScreen:
		if msg.index >= 0 && msg.index < len(m.tables) {
			m.tableIndex = msg.index
			if msg.activate {
				m.loading, m.status, m.err = true, "Opening table", nil
				return m, tea.Batch(m.spinner.Tick, m.tableCmd(m.tables[msg.index].ID))
			}
			return m, nil
		}
		m.tablesActionHover = msg.action
		if !msg.activate {
			return m, nil
		}
		m.tablesActionHover = ""
		if msg.action == "search" {
			m.searchActive, m.searchQuery = true, ""
			return m, nil
		}
		if msg.action == "create" {
			m.screen, m.err = tableCreateScreen, nil
			m.resetTableCreateForm()
			return m, m.form.Init()
		}
		if msg.action == "refresh" {
			m.loading, m.status, m.err = true, "Refreshing tables", nil
			return m, tea.Batch(m.spinner.Tick, m.tablesCmd())
		}
		if msg.action == "back" {
			m.screen, m.err = appMenuScreen, nil
		}
	case tableDetailScreen:
		m.tablesActionHover = msg.action
		if !msg.activate {
			return m, nil
		}
		m.tablesActionHover = ""
		switch msg.action {
		case "record":
			if m.table.CanManage {
				m.startRecordGame()
				return m, nil
			}
		case "refresh":
			m.loading, m.status, m.err = true, "Refreshing table", nil
			return m, tea.Batch(m.spinner.Tick, m.tableCmd(m.table.Table.ID))
		case "back":
			m.screen = tablesScreen
		}
	case formatsScreen:
		if msg.index >= 0 && msg.index < len(m.table.Formats) {
			m.formatIndex = msg.index
			if msg.activate {
				if m.table.CanManage {
					m.screen = formatCreateScreen
					m.resetFormatEditForm()
					return m, m.form.Init()
				}
			}
			return m, nil
		}
		m.formatActionHover = msg.action
		if msg.activate {
			m.formatActionHover = ""
			if msg.action == "search" {
				m.searchActive, m.searchQuery = true, ""
				return m, nil
			}
			if msg.action == "create" && m.table.CanManage {
				m.screen, m.err = formatCreateScreen, nil
				m.resetFormatCreateForm()
				return m, m.form.Init()
			}
			if msg.action == "back" {
				m.screen = tableDetailScreen
			}
		}
	case playersScreen:
		if msg.index >= 0 && msg.index < len(m.table.Players) {
			m.playerIndex = msg.index
			if msg.activate {
				m.screen = playerDetailScreen
				m.resetPlayerEditForm()
				if m.form != nil {
					return m, m.form.Init()
				}
			}
			return m, nil
		}
		m.playerActionHover = msg.action
		if msg.activate {
			m.playerActionHover = ""
			if msg.action == "search" {
				m.searchActive, m.searchQuery = true, ""
				return m, nil
			}
			if msg.action == "create" && m.table.CanManage {
				m.screen, m.err = playerCreateScreen, nil
				m.resetPlayerCreateForm()
				return m, m.form.Init()
			}
			if msg.action == "back" {
				m.screen = tableDetailScreen
			}
		}
	case gamesScreen:
		if msg.index >= 0 && msg.index < len(m.table.Games) {
			m.gameIndex = msg.index
			if msg.activate {
				m.screen, m.resultIndex = gameDetailScreen, 0
			}
			return m, nil
		}
		if msg.action == "search" && msg.activate {
			m.searchActive, m.searchQuery = true, ""
		}
		if msg.action == "record" && msg.activate && m.table.CanManage {
			m.startRecordGame()
			return m, nil
		}
		if msg.action == "back" && msg.activate {
			m.screen = tableDetailScreen
		}
	case gameDetailScreen:
		if msg.action == "search" && msg.activate {
			m.searchActive, m.searchQuery = true, ""
		}
		if msg.action == "edit" && msg.activate && m.canEditSelectedGame() {
			m.startEditSelectedGame()
		}
		if msg.action == "back" && msg.activate {
			m.screen, m.err = gamesScreen, nil
		}
	case recordGameScreen:
		if msg.action != "" && msg.activate {
			switch msg.action {
			case "search":
				if m.recordPhase == recordPlayersPhase {
					m.searchActive, m.searchQuery = true, ""
					return m, nil
				}
			case "metadata":
				if m.recordPhase != recordChipCountsPhase && m.recordPhase != recordDetailsPhase && m.recordPhase != recordReviewPhase {
					return m, m.openRecordMetadataPopup()
				}
			case "back":
				m.screen, m.err = tableDetailScreen, nil
				return m, nil
			case "create":
				if m.recordPhase == recordPlayersPhase && m.table.CanManage {
					m.recordQuickAdd = true
					m.screen, m.err = playerCreateScreen, nil
					m.resetPlayerCreateForm()
					return m, m.form.Init()
				}
			case "delete":
				if m.recordPhase == recordChipCountsPhase && m.currentRecordPlayerIsEntered() {
					if !m.recordClearConfirm {
						m.recordClearConfirm = true
						return m, nil
					}
					m.clearCurrentRecordPlayerEarnings()
				}
				return m, nil
			case "edit":
				if m.recordPhase == recordReviewPhase {
					m.recordPhase, m.recordPreview, m.err = recordPlayersPhase, nil, nil
					return m, nil
				}
				if m.recordPhase == recordPlayersPhase && m.recordPlayerIsEntered() {
					return m, m.beginRecordPlayerEarnings()
				}
				return m, nil
			case "review":
				if m.recordPhase == recordPlayersPhase && m.hasEnoughRecordings() && m.recordHasChanges() {
					m.loading, m.status, m.err = true, "Checking table balance", nil
					return m, tea.Batch(m.spinner.Tick, m.previewRecordGameCmd())
				}
			case "submit", "record":
				if m.recordPhase == recordReviewPhase {
					m.loading, m.status, m.err = true, "Recording game", nil
					return m, tea.Batch(m.spinner.Tick, m.recordGameCmd())
				}
			}
		}
		if m.recordPhase == recordFormatPhase && msg.index >= 0 && msg.index < len(m.table.Formats) {
			m.recordFormatIndex = msg.index
			if msg.activate {
				m.recordPhase = recordPlayersPhase
			}
			return m, nil
		}
		if m.recordPhase == recordPlayersPhase && msg.index >= 0 && msg.index < len(m.table.Players) {
			m.playerIndex = msg.index
			if msg.activate {
				return m, m.beginRecordPlayerEarnings()
			}
			return m, nil
		}
	}
	return m, nil
}

func (m Model) tableHitRegions() []hitRegion {
	if m.loading {
		return nil
	}
	width := max(m.width-4, 44)
	x, y := 2, 1
	var items []actionBarItem
	var actionY, listY int
	regions := []hitRegion{}
	switch m.screen {
	case tablesScreen:
		items = tablesActionItems(true)
		actionY = y + lipgloss.Height(pageHeader(width, "tables")) + 1
		listY = actionY + lipgloss.Height(searchActionBar(items, m.tablesActionHover, m.searchActive, m.searchQuery)) + 1
		regions = actionBarHitRegions(x, actionY, items)
		for row, index := range m.visibleTableIndices() {
			regions = append(regions, hitRegion{x0: x, x1: x + width, y0: listY + 3 + row, y1: listY + 3 + row, value: fmt.Sprintf("table:%d", index)})
		}
	case tableDetailScreen:
		if m.table == nil {
			return nil
		}
		items = tableDetailActionItems(m.table.CanManage)
		actionY = y + lipgloss.Height(m.tableWorkspaceHeader(width)) + 1
		regions = append(regions, m.tableWorkspaceNavRegions(m.tableWorkspaceMetrics(width, items))...)
	case formatsScreen:
		if m.table == nil {
			return nil
		}
		items = formatActionItems(m.table.CanManage)
		actionY = y + lipgloss.Height(m.tableWorkspaceHeader(width)) + 1
		listY = actionY + lipgloss.Height(searchActionBar(items, m.formatActionHover, m.searchActive, m.searchQuery)) + 1
		regions = actionBarHitRegions(x, actionY, items)
		regions = append(regions, m.tableWorkspaceNavRegions(m.tableWorkspaceMetrics(width, items))...)
		for row, index := range m.visibleFormatIndices() {
			regions = append(regions, hitRegion{x0: x, x1: x + width, y0: listY + 7 + row, y1: listY + 7 + row, value: fmt.Sprintf("format:%d", index)})
		}
	case playersScreen:
		if m.table == nil {
			return nil
		}
		items = playerActionItems(m.table.CanManage)
		actionY = y + lipgloss.Height(m.tableWorkspaceHeader(width)) + 1
		listY = actionY + lipgloss.Height(searchActionBar(items, m.playerActionHover, m.searchActive, m.searchQuery)) + 1
		regions = actionBarHitRegions(x, actionY, items)
		regions = append(regions, m.tableWorkspaceNavRegions(m.tableWorkspaceMetrics(width, items))...)
		for row, index := range m.visiblePlayerIndices() {
			regions = append(regions, hitRegion{x0: x, x1: x + width, y0: listY + 7 + row, y1: listY + 7 + row, value: fmt.Sprintf("player:%d", index)})
		}
	case gameDetailScreen:
		if m.table == nil || m.gameIndex < 0 || m.gameIndex >= len(m.table.Games) {
			return nil
		}
		items = []actionBarItem{{key: "/", label: "Search", action: "search"}, {key: "esc", label: "Back", action: "back"}}
		if m.canEditSelectedGame() {
			items = append([]actionBarItem{{key: "e", label: "Edit", action: "edit", accent: true}}, items...)
		}
		actionY = y + lipgloss.Height(pageHeader(width, m.table.Table.Name, "games", m.gameDateLabel(m.gameIndex))) + 1
		regions = actionBarHitRegions(x, actionY, items)
	case gamesScreen:
		if m.table == nil {
			return nil
		}
		items = gamesActionItems(m.table.CanManage)
		actionY = y + lipgloss.Height(m.tableWorkspaceHeader(width)) + 1
		listY = actionY + lipgloss.Height(searchActionBar(gamesActionItems(m.table.CanManage), "", m.searchActive, m.searchQuery)) + 1
		regions = actionBarHitRegions(x, actionY, items)
		regions = append(regions, m.tableWorkspaceNavRegions(m.tableWorkspaceMetrics(width, items))...)
		visible := m.visibleGameIndices()
		for row := range visible {
			index := visible[len(visible)-1-row]
			regions = append(regions, hitRegion{x0: x, x1: x + width, y0: listY + row + 7, y1: listY + row + 7, value: fmt.Sprintf("game:%d", index)})
		}
	case recordGameScreen:
		if m.table == nil {
			return nil
		}
		breadcrumbs := []string{m.table.Table.Name, "record"}
		if m.recordDetails != nil && strings.TrimSpace(m.recordDetails.date) != "" {
			breadcrumbs = append(breadcrumbs, m.recordDetails.date)
		}
		actionY := y + lipgloss.Height(pageHeader(width, breadcrumbs...)) + 1
		regions = append(regions, actionBarHitRegions(x, actionY, m.recordActionItems())...)
		baseY := actionY + 1
		if m.recordPhase == recordFormatPhase {
			for index := range m.table.Formats {
				itemY := baseY + 2 + index*3
				regions = append(regions, hitRegion{x0: x, x1: x + width, y0: itemY, y1: itemY + 1, value: fmt.Sprintf("format:%d", index)})
			}
		}
		if m.recordPhase == recordPlayersPhase {
			balanceOffset := lipgloss.Height(m.recordBalanceStrip(width)) + 1
			for row, index := range m.visiblePlayerIndices() {
				itemY := baseY + balanceOffset + 2 + row*3
				regions = append(regions, hitRegion{x0: x, x1: x + width, y0: itemY, y1: itemY + 1, value: fmt.Sprintf("player:%d", index)})
			}
		}
	}
	return regions
}

func tablesActionItems(canCreate bool) []actionBarItem {
	items := []actionBarItem{
		{key: "r", label: "Refresh", action: "refresh"},
		{key: "/", label: "Search", action: "search"},
		{key: "esc", label: "Back", action: "back"},
	}
	if canCreate {
		items = append([]actionBarItem{{key: "c", label: "Create table", action: "create", accent: true}}, items...)
	}
	return items
}

func tableDetailActionItems(canManage bool) []actionBarItem {
	items := []actionBarItem{
		{key: "r", label: "Refresh", action: "refresh"},
		{key: "esc", label: "Back", action: "back"},
	}
	if canManage {
		items = append([]actionBarItem{{key: "c", label: "Record game", action: "record", accent: true}}, items...)
	}
	return items
}

func formatActionItems(canManage bool) []actionBarItem {
	items := []actionBarItem{{key: "/", label: "Search", action: "search"}, {key: "esc", label: "Back", action: "back"}}
	if canManage {
		items = append([]actionBarItem{{key: "c", label: "Create format", action: "create", accent: true}}, items...)
	}
	return items
}

func playerActionItems(canManage bool) []actionBarItem {
	items := []actionBarItem{{key: "/", label: "Search", action: "search"}, {key: "esc", label: "Back", action: "back"}}
	if canManage {
		items = append([]actionBarItem{{key: "c", label: "Add player", action: "create", accent: true}}, items...)
	}
	return items
}

func (m Model) tablesView() string {
	width := max(m.width-4, 44)
	if m.loading {
		return m.pageView(lipgloss.JoinVertical(lipgloss.Left, pageHeader(width, "tables"), "", m.spinner.View()+"  "+valueStyle.Render(m.status)), tablesFooter())
	}
	items := tablesActionItems(true)
	parts := []string{pageHeader(width, "tables"), "", searchActionBar(items, m.tablesActionHover, m.searchActive, m.searchQuery), m.tableSummaryList(width)}
	if m.err != nil {
		parts = append(parts, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	return m.pageView(strings.Join(parts, "\n"), tablesFooter())
}

func tablesFooter() string {
	return "↑↓ move   enter open   c create"
}

func (m Model) tableSummaryList(width int) string {
	if len(m.tables) == 0 {
		return spacedEmptyState(lipgloss.JoinVertical(lipgloss.Left, valueStyle.Render("No tables yet"), mutedStyle.Render("Create a table to start a private ledger.")))
	}
	if len(m.visibleTableIndices()) == 0 {
		return spacedEmptyState(mutedStyle.Render("No tables match the search."))
	}
	columns := bluffTableColumns(width, []string{"TABLE", "HOST", "PLAYERS", "GAMES", "LAST GAME"}, 30, 24, 14, 14, 18)
	visible := m.visibleTableIndices()
	rows := make([]bubblesTable.Row, 0, len(visible))
	for _, index := range visible {
		table := m.tables[index]
		selected := index == m.tableIndex
		lastGame := "—"
		if table.LastGameDate != nil {
			lastGame = *table.LastGameDate
		}
		host := "@" + truncate(table.HostUsername, 11) + " " + lipgloss.NewStyle().Foreground(colorFuchsia).Render("♛")
		rows = append(rows, bluffTableRow(selected,
			truncate(table.Name, max(columns[0].Width, 1)),
			host,
			strconv.Itoa(table.PlayerCount),
			strconv.Itoa(table.GameCount),
			lastGame,
		))
	}
	table := newBluffTable(columns, rows, width, m.listTableHeight(10, len(rows)))
	setBluffTableCursor(&table, tableSelectedRow(visible, m.tableIndex, false))
	return bluffTableView(table)
}

func weightedGridColumns(width int, weights ...int) []int {
	if len(weights) == 0 {
		return nil
	}
	content := max(width, len(weights))
	total := 0
	for _, weight := range weights {
		total += max(weight, 0)
	}
	if total == 0 {
		total = len(weights)
	}
	columns := make([]int, len(weights))
	used := 0
	for index, weight := range weights {
		if index == len(weights)-1 {
			columns[index] = max(content-used, 1)
			break
		}
		columns[index] = max(content*max(weight, 0)/total, 1)
		used += columns[index]
	}
	return columns
}

func (m Model) tableDetailView() string {
	width := max(m.width-4, 44)
	if m.loading && m.table == nil {
		return m.pageView(lipgloss.JoinVertical(lipgloss.Left, pageHeader(width, "tables"), "", m.spinner.View()+"  "+valueStyle.Render(m.status)), tableDetailFooter())
	}
	if m.table == nil {
		return m.pageView(pageHeader(width, "tables"), tableDetailFooter())
	}
	items := tableDetailActionItems(m.table.CanManage)
	metrics := m.tableWorkspaceMetrics(width, items)
	content := m.tableOverviewContent(metrics.width)
	if m.loading {
		content = m.spinner.View() + "  " + valueStyle.Render(m.status)
	}
	if m.notice != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", lipgloss.NewStyle().Foreground(colorGreen).Render("✓ "+m.notice))
	}
	footer := tableDetailFooter()
	if m.expandedChart != tableChartNone {
		footer = "esc back"
		if m.expandedChart == tableChartStandings {
			footer = m.standingEntryShortcuts()
		}
	}
	return m.tableWorkspace(tableOverviewSection, items, content, footer)
}

func tableDetailFooter() string {
	return "tab next   shift+tab previous   esc back"
}

func (m Model) tableStandings(width int) string {
	lines := []string{sectionHeading("Standings", width)}
	if len(m.table.Players) == 0 {
		return strings.Join(append(lines, mutedStyle.Render("Add players when you are ready to record a game.")), "\n")
	}
	for index, playerIndex := range m.visiblePlayerIndices() {
		player := m.table.Players[playerIndex]
		amount := standingStyle(player.Standing).Render(signedCredits(player.Standing))
		name := valueStyle.Render(truncate(displayTablePlayerName(player, m.table.Table.HostUsername), max(width-20, 12)))
		gap := max(width-lipgloss.Width(name)-lipgloss.Width(amount)-8, 1)
		lines = append(lines, fmt.Sprintf("%2d  %s%s%s", index+1, name, strings.Repeat(" ", gap), amount))
	}
	return strings.Join(lines, "\n")
}

func (m Model) tableRecentGames(width int) string {
	lines := []string{sectionHeading("Recent games", width)}
	if len(m.table.Games) == 0 {
		return strings.Join(append(lines, mutedStyle.Render("Completed games will appear here.")), "\n")
	}
	visible := m.visibleGameIndices()
	start := max(len(visible)-5, 0)
	for row := len(visible) - 1; row >= start; row-- {
		index := visible[row]
		game := m.table.Games[index]
		label := valueStyle.Render(truncate(game.Format.Name, max(width-34, 10)))
		meta := mutedStyle.Render(fmt.Sprintf("%s  ·  %d players", game.Date, len(game.Participants)))
		gap := max(width-lipgloss.Width(label)-lipgloss.Width(meta)-lipgloss.Width(statusBadge(game.Status))-4, 1)
		lines = append(lines, label+strings.Repeat(" ", gap)+statusBadge(game.Status), meta)
	}
	return strings.Join(lines, "\n")
}

func (m Model) formatsView() string {
	width := max(m.width-4, 44)
	if m.table == nil {
		return m.pageView(pageHeader(width, "tables", "formats"), formatsFooter(m.table != nil && m.table.CanManage))
	}
	items := formatActionItems(m.table.CanManage)
	metrics := m.tableWorkspaceMetrics(width, items)
	content := m.formatList(workspaceContentWidth(metrics))
	if m.loading {
		content = m.spinner.View() + "  " + valueStyle.Render(m.status)
	}
	return m.tableWorkspace(tableFormatsSection, items, content, formatsFooter(m.table.CanManage))
}

func formatsFooter(canManage bool) string {
	if canManage {
		return "↑↓ move   enter edit   c create"
	}
	return "↑↓ move"
}

func (m Model) formatList(width int) string {
	if len(m.table.Formats) == 0 {
		return workspaceEmptyState(width, "No game formats yet", "The table host can add the first format.")
	}
	if len(m.visibleFormatIndices()) == 0 {
		return spacedEmptyState(lipgloss.NewStyle().Width(width).Render(mutedStyle.Render("No formats match the search.")))
	}
	// Keep entry close to the format name and give chips most of the row so
	// wider terminals can show the complete denomination set.
	columns := bluffTableColumns(width, []string{"FORMAT", "ENTRY", "CHIPS"}, 38, 14, 48)
	visible := m.visibleFormatIndices()
	rows := make([]bubblesTable.Row, 0, len(visible))
	for _, index := range visible {
		format := m.table.Formats[index]
		selected := index == m.formatIndex
		chipValues := make([]string, 0, len(format.Chips))
		for _, chip := range format.Chips {
			chipValues = append(chipValues, fmt.Sprintf("%s %d", chipSwatch(chip.Color), chip.Value))
		}
		rows = append(rows, bluffTableRow(selected,
			truncate(format.Name, max(columns[0].Width, 1)),
			credits(format.RequiredEntry),
			strings.Join(chipValues, "  "),
		))
	}
	table := newBluffTable(columns, rows, width, m.listTableHeight(10, len(rows)))
	setBluffTableCursor(&table, tableSelectedRow(visible, m.formatIndex, false))
	return bluffTableView(table)
}

func (m Model) playersView() string {
	width := max(m.width-4, 44)
	if m.table == nil {
		return m.pageView(pageHeader(width, "tables", "players"), playersFooter(false))
	}
	items := playerActionItems(m.table.CanManage)
	metrics := m.tableWorkspaceMetrics(width, items)
	content := m.playerList(workspaceContentWidth(metrics))
	if m.notice != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", lipgloss.NewStyle().Foreground(colorGreen).Render("✓ "+m.notice))
	}
	if m.loading {
		content = m.spinner.View() + "  " + valueStyle.Render(m.status)
	}
	return m.tableWorkspace(tablePlayersSection, items, content, playersFooter(m.table.CanManage))
}

func playersFooter(canManage bool) string {
	if canManage {
		return "↑↓ move   c add player"
	}
	return "↑↓ move"
}

func (m Model) gamesView() string {
	width := max(m.width-4, 44)
	if m.table == nil {
		return m.pageView(pageHeader(width, "tables", "games"), "")
	}
	items := gamesActionItems(m.table.CanManage)
	metrics := m.tableWorkspaceMetrics(width, items)
	content := m.gameList(workspaceContentWidth(metrics))
	if m.loading {
		content = m.spinner.View() + "  " + valueStyle.Render(m.status)
	}
	return m.tableWorkspace(tableGamesSection, items, content, gamesFooter(m.table.CanManage))
}

func gamesFooter(canManage bool) string {
	if canManage {
		return "↑↓ move   enter inspect   c record"
	}
	return "↑↓ move   enter inspect"
}

func gamesActionItems(canManage bool) []actionBarItem {
	items := []actionBarItem{{key: "/", label: "Search", action: "search"}, {key: "esc", label: "Back", action: "back"}}
	if canManage {
		items = append([]actionBarItem{{key: "c", label: "Record game", action: "record", accent: true}}, items...)
	}
	return items
}

func (m Model) gameList(width int) string {
	if len(m.table.Games) == 0 {
		return workspaceEmptyState(width, "No games recorded yet", "Recorded games will stay here with their chip snapshots.")
	}
	if len(m.visibleGameIndices()) == 0 {
		return spacedEmptyState(lipgloss.NewStyle().Width(width).Render(mutedStyle.Render("No games match the search.")))
	}
	columns := bluffTableColumns(width, []string{"DATE", "FORMAT", "PLAYERS", "STATUS"}, 18, 42, 15, 25)
	visible := m.visibleGameIndices()
	rows := make([]bubblesTable.Row, 0, len(visible))
	for row := len(visible) - 1; row >= 0; row-- {
		index := visible[row]
		game := m.table.Games[index]
		selected := index == m.gameIndex
		rows = append(rows, bluffTableRow(selected,
			truncate(m.gameDateLabel(index), max(columns[0].Width, 1)),
			truncate(game.Format.Name, max(columns[1].Width, 1)),
			strconv.Itoa(len(game.Participants)),
			statusBadge(game.Status),
		))
	}
	table := newBluffTable(columns, rows, width, m.listTableHeight(10, len(rows)))
	setBluffTableCursor(&table, tableSelectedRow(visible, m.gameIndex, true))
	return bluffTableView(table)
}

func (m Model) gameDetailView() string {
	width := max(m.width-4, 44)
	if m.table == nil || m.gameIndex < 0 || m.gameIndex >= len(m.table.Games) {
		return m.pageView(pageHeader(width, "tables", "games"), "")
	}
	game := m.table.Games[m.gameIndex]
	actions := []actionBarItem{{key: "/", label: "Search", action: "search"}, {key: "esc", label: "Back", action: "back"}}
	if m.canEditSelectedGame() {
		actions = append([]actionBarItem{{key: "e", label: "Edit", action: "edit", accent: true}}, actions...)
	}
	body := gameSummaryBody(
		m.gameInspectionBalanceStrip(game, width),
		game.Remarks,
		m.gameInspectionPlayerList(game, width),
		width,
	)
	content := lipgloss.JoinVertical(lipgloss.Left,
		pageHeader(width, m.table.Table.Name, "games", m.gameDateLabel(m.gameIndex)),
		"",
		searchActionBar(actions, "", m.searchActive, m.searchQuery),
		body,
	)
	return m.pageView(content, "↑↓ move")
}

func (m Model) gameDateLabel(index int) string {
	if m.table == nil || index < 0 || index >= len(m.table.Games) {
		return ""
	}
	date := m.table.Games[index].Date
	count, sequence := 0, 0
	for gameIndex, game := range m.table.Games {
		if game.Date != date {
			continue
		}
		count++
		if gameIndex <= index {
			sequence++
		}
	}
	if count < 2 {
		return date
	}
	return fmt.Sprintf("%s [%d]", date, sequence)
}

func (m Model) gameInspectionPlayerList(game api.TableGame, width int) string {
	participants := append([]api.TableGameParticipant(nil), game.Participants...)
	slices.SortStableFunc(participants, func(left, right api.TableGameParticipant) int {
		switch {
		case left.ProfitLoss > right.ProfitLoss:
			return -1
		case left.ProfitLoss < right.ProfitLoss:
			return 1
		default:
			return strings.Compare(strings.ToLower(left.PlayerName), strings.ToLower(right.PlayerName))
		}
	})
	items := make([]list.Item, 0, len(participants))
	for _, participant := range participants {
		name := displayParticipantName(participant.PlayerName, m.table.Table.HostUsername)
		if !searchMatches(m.searchQuery, name) && !searchMatches(m.searchQuery, participant.PlayerName) {
			continue
		}
		description := valueStyle.Render("Total "+credits(participant.FinalValue)) + "  ·  " +
			standingStyle(participant.ProfitLoss).Render("P/L "+signedCredits(participant.ProfitLoss))
		if participant.FinalValue == 0 {
			description = lipgloss.NewStyle().Bold(true).Foreground(colorRed).Render("ALL IN")
		} else if chips := participantChipCountLine(participant.ChipCounts); chips != "" {
			description += "  ·  " + chips
		}
		items = append(items, bluffListItem{title: name, description: description})
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#071A14")).
		Background(lipgloss.Color("#FFD866")).Padding(0, 1)
	selected := min(max(m.resultIndex, 0), max(len(items)-1, 0))
	return bluffListViewWithTitleStyle("Inspecting", items, selected, width, m.recordListHeight(len(items)), &titleStyle)
}

func (m Model) inspectionPlayerCount() int {
	if m.table == nil || m.gameIndex < 0 || m.gameIndex >= len(m.table.Games) {
		return 0
	}
	count := 0
	for _, participant := range m.table.Games[m.gameIndex].Participants {
		name := displayParticipantName(participant.PlayerName, m.table.Table.HostUsername)
		if searchMatches(m.searchQuery, name) || searchMatches(m.searchQuery, participant.PlayerName) {
			count++
		}
	}
	return count
}

func recordChipCountLine(chips []api.ChipDenomination, counts map[string]int) string {
	values := make([]string, 0, len(chips))
	for _, chip := range chips {
		if count := counts[chip.ID]; count > 0 {
			values = append(values, fmt.Sprintf("%s × %d", chipSwatch(chip.Color), count))
		}
	}
	return strings.Join(values, "  ")
}

func participantChipCountLine(chips []api.ChipCount) string {
	values := make([]string, 0, len(chips))
	for _, chip := range chips {
		if chip.Count > 0 {
			values = append(values, fmt.Sprintf("%s × %d", chipSwatch(chip.Color), chip.Count))
		}
	}
	return strings.Join(values, "  ")
}

func (m *Model) moveResultSelection(delta, count int) {
	if count <= 0 {
		m.resultIndex = 0
		return
	}
	m.resultIndex = max(0, min(m.resultIndex+delta, count-1))
}

func (m Model) gameInspectionBalanceStrip(game api.TableGame, width int) string {
	deficit := game.ExpectedTableValue - game.ActualTableValue
	balance := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#071A14")).
		Background(colorGreen).Padding(0, 1).Render("BALANCED")
	if deficit != 0 {
		balance = lipgloss.NewStyle().Bold(true).Foreground(colorRed).Render(signedCredits(deficit))
	}
	stats := []string{
		workspaceStat("FORMAT", game.Format.Name),
		workspaceStat("PLAYERS", strconv.Itoa(len(game.Participants))),
		workspaceStat("TABLE VALUE", credits(game.ExpectedTableValue)),
		workspaceStat("RECORDED VALUE", credits(game.ActualTableValue)),
		workspaceStat("DEFICIT", balance),
	}
	columnWidth := max((width-14)/5, 9)
	columns := make([]string, 0, len(stats))
	for _, stat := range stats {
		columns = append(columns, lipgloss.NewStyle().Width(columnWidth).Align(lipgloss.Center).Render(stat))
	}
	content := lipgloss.JoinHorizontal(lipgloss.Top, columns[0], "  ", columns[1], "  ", columns[2], "  ", columns[3], "  ", columns[4])
	return lipgloss.NewStyle().Width(width).Padding(0, 1).
		Border(lipgloss.NormalBorder()).BorderForeground(colorMuted).Render(content)
}

func (m Model) playerList(width int) string {
	if len(m.table.Players) == 0 {
		return workspaceEmptyState(width, "No players yet", "Add a player before recording a game.")
	}
	if len(m.visiblePlayerIndices()) == 0 {
		return spacedEmptyState(lipgloss.NewStyle().Width(width).Render(mutedStyle.Render("No players match the search.")))
	}
	columns := bluffTableColumns(width, []string{"PLAYER", "STANDING"}, 3, 1)
	visible := m.visiblePlayerIndices()
	rows := make([]bubblesTable.Row, 0, len(visible))
	for _, index := range visible {
		player := m.table.Players[index]
		selected := index == m.playerIndex
		rows = append(rows, bluffTableRow(selected,
			truncate(displayTablePlayerName(player, m.table.Table.HostUsername), max(columns[0].Width, 1)),
			signedCreditNumber(player.Standing),
		))
	}
	table := newBluffTable(columns, rows, width, m.listTableHeight(10, len(rows)))
	setBluffTableCursor(&table, tableSelectedRow(visible, m.playerIndex, false))
	return bluffTableView(table)
}

func signedCreditNumber(value int) string {
	if value > 0 {
		return "+" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func (m Model) tableCreateView() string {
	return m.centeredFormPage("tables", "Create a table", "Create", "")
}

func (m Model) formatCreateView() string {
	if m.table == nil {
		return m.pageView(pageHeader(max(m.width-4, 44), "formats"), "")
	}
	background := m
	background.screen = formatsScreen
	background.err = nil
	title, actionLabel, footer := "Create game format", "Create", "tab next   enter create   esc close"
	if m.formatEditIndex >= 0 {
		title, actionLabel, footer = "Edit game format", "Save", "tab next   enter save   esc close"
	}
	return m.formPopupSizedWithActions(background.formatsView(), title, m.popupFormView(54), footer, 54, []actionBarItem{
		{key: "enter", label: actionLabel, action: "submit", accent: true},
		{key: "esc", label: "Close", action: "close"},
	})
}

func (m Model) playerCreateView() string {
	if m.table == nil {
		return m.pageView(pageHeader(max(m.width-4, 44), "players"), "")
	}
	background := m
	background.screen = playersScreen
	background.err = nil
	return m.formPopupWithActions(background.playersView(), "Add player", m.popupFormView(72), "tab next   enter add   esc close", []actionBarItem{
		{key: "enter", label: "Add", action: "submit", accent: true},
		{key: "esc", label: "Close", action: "close"},
	})
}

func (m Model) playerDetailView() string {
	if m.table == nil || m.playerIndex < 0 || m.playerIndex >= len(m.table.Players) {
		return m.pageView(pageHeader(max(m.width-4, 44), "players"), "")
	}
	player := m.table.Players[m.playerIndex]
	background := m
	background.screen = playersScreen
	background.err = nil
	inviteCode := m.playerInviteCodeForCurrentPlayer()
	if m.playerInvitePopup && inviteCode != "" {
		body := lipgloss.JoinVertical(lipgloss.Left,
			valueStyle.Render("Invite code"),
			brandStyle.Render(inviteCode),
			"",
			mutedStyle.Render("Share this code once. It can be used to create one account."),
		)
		return m.formPopupWithActions(background.playersView(), "Invite code", body, "esc close", []actionBarItem{
			{key: "esc", label: "Close", action: "close"},
		})
	}

	details := []string{}
	if inviteCode != "" {
		details = append(details, mutedStyle.Render("Invite code"), brandStyle.Render(inviteCode))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, details...)
	if m.form != nil && m.playerCanEdit() {
		formView := m.form.View()
		if body == "" {
			body = formView
		} else {
			body = lipgloss.JoinVertical(lipgloss.Left, formView, "", body)
		}
	} else if strings.TrimSpace(player.Username) != "" {
		body = lipgloss.JoinVertical(lipgloss.Left,
			mutedStyle.Render("Username"),
			valueStyle.Render("@"+player.Username),
			"", body,
		)
	}

	popupActions := []actionBarItem{{key: "esc", label: "Close", action: "close"}}
	footer := "esc close"
	if m.table.CanManage {
		if m.playerCanEdit() {
			popupActions = append([]actionBarItem{{key: "enter", label: "Save", action: "submit", accent: true}}, popupActions...)
			footer = "enter save   esc close"
			if inviteCode == "" {
				popupActions = append([]actionBarItem{{key: "c", label: "Create invite", action: "invite", accent: true}}, popupActions...)
				footer = "enter save   c invite   esc close"
			}
		}
		if m.playerHasEntries() {
			popupActions = append([]actionBarItem{{key: "x", label: "Disable", action: "disable", accent: true}}, popupActions...)
			footer = "x disable   " + footer
		} else {
			deleteLabel := "Delete"
			deleteHelp := "alt+d delete"
			if m.playerDeleteConfirm {
				deleteLabel = "Confirm Delete"
				deleteHelp = "alt+d confirm delete"
			}
			popupActions = append([]actionBarItem{{key: "alt+d", label: deleteLabel, action: "delete"}}, popupActions...)
			footer = deleteHelp + "   " + footer
		}
	}
	headerStanding := standingStyle(player.Standing).Render(signedCredits(player.Standing))
	return m.formPopupSizedWithHeader(background.playersView(), "Player", headerStanding, body, footer, 72, popupActions)
}

func (m Model) playerCanEdit() bool {
	if m.table == nil || m.playerIndex < 0 || m.playerIndex >= len(m.table.Players) {
		return false
	}
	return m.table.CanManage && strings.TrimSpace(m.table.Players[m.playerIndex].Username) == ""
}

func (m Model) playerInviteCodeForCurrentPlayer() string {
	if m.table == nil || m.playerIndex < 0 || m.playerIndex >= len(m.table.Players) {
		return ""
	}
	if m.playerInviteCodes == nil {
		return ""
	}
	return m.playerInviteCodes[m.table.Players[m.playerIndex].ID]
}

// formPopup composes a centered modal over the current page. The page remains
// visible, which makes the modal feel like a focused action instead of a new
// route.
func (m Model) formPopup(background, title, content, footer string) string {
	return m.formPopupWithActions(background, title, content, footer, nil)
}

func (m Model) formPopupSized(background, title, content, footer string, maxWidth int) string {
	return m.formPopupSizedWithActions(background, title, content, footer, maxWidth, nil)
}

func popupWidth(terminalWidth, maxWidth int) int {
	return min(max(terminalWidth-12, 38), maxWidth)
}

// popupFormView sizes Huh to the popup's inner width before rendering. Without
// this, a long field underline wraps inside the modal and looks like a second
// divider.
func (m Model) popupFormView(maxWidth int) string {
	if m.form == nil {
		return ""
	}
	width := popupWidth(m.width, maxWidth)
	m.form.WithWidth(max(width-8, 20))
	return m.form.View()
}

func (m Model) formPopupWithActions(background, title, content, footer string, actions []actionBarItem) string {
	return m.formPopupSizedWithActions(background, title, content, footer, 72, actions)
}

func (m Model) formPopupSizedWithActions(background, title, content, footer string, maxWidth int, actions []actionBarItem) string {
	return m.formPopupSizedWithHeader(background, title, "", content, footer, maxWidth, actions)
}

func (m Model) formPopupSizedWithHeader(background, title, headerRight, content, footer string, maxWidth int, actions []actionBarItem) string {
	width := popupWidth(m.width, maxWidth)
	if len(actions) == 0 {
		actions = []actionBarItem{{key: "esc", label: "Close", action: "close"}}
	}
	if m.err != nil {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	contentWidth := max(width-8, 20)
	popupFooter := popupActionFooter(actions, contentWidth)
	popup := lipgloss.NewStyle().Width(width).Padding(0, 3).
		Border(lipgloss.RoundedBorder()).BorderForeground(colorIndigo).
		Align(lipgloss.Left).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			popupHeaderWithRight(title, headerRight, contentWidth),
			"", "",
			content,
			"", "",
			popupFooter,
		))
	return overlayPage(background, popup, m.width, m.height, m.helpBar(footer))
}

func overlayPage(background, popup string, width, height int, footer string) string {
	if width <= 0 || height <= 1 {
		return popup + "\n" + footer
	}
	bodyHeight := height - 1
	baseLines := strings.Split(background, "\n")
	if len(baseLines) > bodyHeight {
		baseLines = baseLines[:bodyHeight]
	}
	for len(baseLines) < bodyHeight {
		baseLines = append(baseLines, "")
	}
	popupLines := strings.Split(popup, "\n")
	popupWidth := lipgloss.Width(popup)
	popupHeight := len(popupLines)
	x := max((width-popupWidth)/2, 0)
	y := max((bodyHeight-popupHeight)/2, 0)
	for row, popupLine := range popupLines {
		if y+row >= len(baseLines) {
			break
		}
		baseLine := padTerminalLine(baseLines[y+row], width)
		left := ansi.Cut(baseLine, 0, x)
		right := ansi.Cut(baseLine, min(x+popupWidth, width), width)
		popupLine = padTerminalLine(popupLine, popupWidth)
		baseLines[y+row] = left + "\x1b[0m" + popupLine + "\x1b[0m" + right
	}
	return strings.Join(baseLines, "\n") + "\n" + footer
}

func padTerminalLine(line string, width int) string {
	missing := width - ansi.StringWidth(line)
	if missing <= 0 {
		return line
	}
	return line + strings.Repeat(" ", missing)
}

func (m Model) centeredFormPage(crumb, title, _ string, footer string) string {
	width := max(m.width-4, 44)
	body := lipgloss.JoinVertical(lipgloss.Center, pageHeader(width, strings.Split(crumb, " / ")...), "", brandStyle.Render(title), "", m.form.View())
	if m.err != nil {
		body = lipgloss.JoinVertical(lipgloss.Center, body, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	return m.pageView(lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(body), footer)
}

func (m Model) recordGameView() string {
	width := max(m.width-4, 44)
	if m.table == nil {
		return m.pageView(pageHeader(width, "record game"), "")
	}
	breadcrumbs := []string{m.table.Table.Name, "record"}
	if m.recordDetails != nil && strings.TrimSpace(m.recordDetails.date) != "" {
		breadcrumbs = append(breadcrumbs, m.recordDetails.date)
	}
	header := pageHeader(width, breadcrumbs...)
	if m.loading {
		return m.pageView(lipgloss.JoinVertical(lipgloss.Left, header, "", m.spinner.View()+"  "+valueStyle.Render(m.status)), "please wait")
	}
	if m.recordPopup != recordPopupNone {
		background := m
		background.form = nil
		background.err = nil
		background.recordPopup = recordPopupNone
		// An earnings view requires its counter form. Metadata overlays remove
		// that form from the background so two popups are never composited. If
		// an input event opens metadata while earnings is active, fall back to
		// the recorder's player list instead of dereferencing a nil Huh form.
		if background.recordPhase == recordChipCountsPhase {
			background.recordPhase = recordPlayersPhase
		}
		return m.formPopupSizedWithActions(background.recordGameView(), "Edit date/note", m.popupFormView(58), "enter save   esc close", 58, []actionBarItem{
			{key: "enter", label: "Save", action: "submit", accent: true},
			{key: "esc", label: "Close", action: "close"},
		})
	}
	var body string
	switch m.recordPhase {
	case recordDetailsPhase:
		body = mutedStyle.Render("Choose a game format to continue.")
	case recordFormatPhase:
		body = m.recordFormatList(width)
	case recordPlayersPhase:
		body = lipgloss.JoinVertical(lipgloss.Left, m.recordBalanceStrip(width), "", m.recordPlayerList(width))
	case recordChipCountsPhase:
		background := m
		background.recordPhase = recordPlayersPhase
		background.form = nil
		background.err = nil
		background.recordPopup = recordPopupNone
		player := m.table.Players[m.recordPlayerIndex]
		popupTitle := "Player earnings · " + displayTablePlayerName(player, m.table.Table.HostUsername)
		popupActions := []actionBarItem{
			{key: "enter", label: "Save", action: "submit", accent: true},
			{key: "esc", label: "Close", action: "close"},
		}
		if m.currentRecordPlayerIsEntered() {
			clearLabel := "Clear earning"
			if m.recordClearConfirm {
				clearLabel = "Confirm Clear"
			}
			popupActions = append([]actionBarItem{{key: "alt+d", label: clearLabel, action: "delete"}}, popupActions...)
		}
		return m.formPopupSizedWithActions(background.recordGameView(), popupTitle, m.recordChipCountView(), recordFooter(m.recordPhase, m.recordPreview != nil), 58, popupActions)
	case recordReviewPhase:
		note := ""
		if m.recordDetails != nil {
			note = m.recordDetails.remarks
		}
		body = gameSummaryBody(m.recordBalanceStrip(width), note, m.recordReviewPlayerList(width), width)
	}
	if m.err != nil {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	actionItems := m.recordActionItems()
	actions := actionBar(actionItems, "")
	if m.recordPhase == recordPlayersPhase {
		actions = searchActionBar(actionItems, "", m.searchActive, m.searchQuery)
	}
	return m.pageView(lipgloss.JoinVertical(lipgloss.Left, header, "", actions, body), recordFooter(m.recordPhase, m.recordPreview != nil))
}

func gameSummaryBody(stats, note, players string, width int) string {
	parts := []string{stats}
	if noteView := gameNoteView(note, width); noteView != "" {
		parts = append(parts, "", noteView)
	}
	parts = append(parts, "", players)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func gameNoteView(note string, width int) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return ""
	}
	contentWidth := max(width-2, 1)
	wrapped := ansi.Wordwrap(note, contentWidth, "")
	content := lipgloss.JoinVertical(lipgloss.Left, brandStyle.Render("Note"), valueStyle.Render(wrapped))
	return lipgloss.NewStyle().Width(width).Padding(0, 1).Render(content)
}

func (m Model) recordActionItems() []actionBarItem {
	if m.recordPhase == recordReviewPhase {
		return []actionBarItem{
			{key: "s", label: "Submit", action: "submit", accent: true},
			{key: "e", label: "Edit", action: "edit"},
			{key: "esc", label: "Back", action: "back"},
		}
	}
	items := []actionBarItem{{key: "esc", label: "Back", action: "back"}}
	if m.recordPhase != recordChipCountsPhase && m.recordPhase != recordDetailsPhase {
		items = append([]actionBarItem{
			{key: "t", label: "Edit date/note", action: "metadata"},
		}, items...)
	}
	if m.recordPhase == recordPlayersPhase && m.table.CanManage {
		items = append([]actionBarItem{
			{key: "/", label: "Search", action: "search"},
		}, items...)
		if m.recordEditGameID == "" {
			items = append([]actionBarItem{{key: "c", label: "Add player", action: "create", accent: true}}, items...)
		}
		if m.recordPlayerIsEntered() {
			items = append([]actionBarItem{{key: "e", label: "Edit earning", action: "edit"}}, items...)
		}
		if m.hasEnoughRecordings() && m.recordHasChanges() {
			items = append([]actionBarItem{{key: "r", label: "Review game", action: "review", accent: true}}, items...)
		}
	}
	return items
}

func recordFooter(phase recordPhase, _ bool) string {
	switch phase {
	case recordDetailsPhase:
		return "t edit date/note"
	case recordFormatPhase:
		return "↑↓ move   enter choose format   t edit date/note"
	case recordPlayersPhase:
		return "↑↓ move   enter add/edit earnings   t edit date/note"
	case recordChipCountsPhase:
		return "↑↓ move   ←→ adjust / all in   shift+←→ ±10   enter save   esc close"
	case recordReviewPhase:
		return "↑↓ move   s submit   e edit   esc back"
	default:
		return ""
	}
}

func (m Model) recordFormatList(width int) string {
	if len(m.table.Formats) == 0 {
		return workspaceEmptyState(width, "No game formats yet", "Create a game format before recording a game.")
	}
	items := make([]list.Item, 0, len(m.table.Formats))
	for _, format := range m.table.Formats {
		chips := make([]string, 0, len(format.Chips))
		for _, chip := range format.Chips {
			chips = append(chips, fmt.Sprintf("%s %d", chipSwatch(chip.Color), chip.Value))
		}
		description := "Buy-in " + credits(format.RequiredEntry)
		if len(chips) > 0 {
			description += "  ·  " + strings.Join(chips, "  ")
		}
		items = append(items, bluffListItem{title: format.Name, description: description})
	}
	return bluffListView("Choose a game format", items, m.recordFormatIndex, width, m.recordListHeight(len(items)))
}

func (m Model) recordPlayerList(width int) string {
	visible := m.visiblePlayerIndices()
	if m.recordEditGameID != "" {
		visible = slices.DeleteFunc(visible, func(index int) bool {
			return !m.recordEntered[m.table.Players[index].ID]
		})
	}
	items := make([]list.Item, 0, len(visible))
	for _, index := range visible {
		player := m.table.Players[index]
		name := displayTablePlayerName(player, m.table.Table.HostUsername)
		description := mutedStyle.Render("No earning added")
		if m.recordPlayerAllIn(player.ID) {
			description = lipgloss.NewStyle().Bold(true).Foreground(colorRed).Render("ALL IN")
		} else if total, ok := m.recordPlayerTotal(player.ID); ok {
			pnl := total - m.table.Formats[m.recordFormatIndex].RequiredEntry
			description = valueStyle.Render("Total "+credits(total)) + "  ·  " + standingStyle(pnl).Render("P/L "+signedCredits(pnl))
		}
		items = append(items, bluffListItem{title: name, description: description})
	}
	selected := tableSelectedRow(visible, m.playerIndex, false)
	return bluffListView("Add/Edit earnings", items, selected, width, m.recordListHeight(len(items)))
}

func (m Model) recordReviewPlayerList(width int) string {
	type rankedPlayer struct {
		item list.Item
		pnl  int
	}
	ranked := make([]rankedPlayer, 0, len(m.recordEntered))
	for _, player := range m.table.Players {
		if !m.recordEntered[player.ID] {
			continue
		}
		name := displayTablePlayerName(player, m.table.Table.HostUsername)
		pnl := -m.table.Formats[m.recordFormatIndex].RequiredEntry
		description := lipgloss.NewStyle().Bold(true).Foreground(colorRed).Render("ALL IN")
		if !m.recordPlayerAllIn(player.ID) {
			total, _ := m.recordPlayerTotal(player.ID)
			pnl = total - m.table.Formats[m.recordFormatIndex].RequiredEntry
			description = valueStyle.Render("Total "+credits(total)) + "  ·  " + standingStyle(pnl).Render("P/L "+signedCredits(pnl))
			if chips := recordChipCountLine(m.table.Formats[m.recordFormatIndex].Chips, m.recordCounts[player.ID]); chips != "" {
				description += "  ·  " + chips
			}
		}
		ranked = append(ranked, rankedPlayer{item: bluffListItem{title: name, description: description}, pnl: pnl})
	}
	slices.SortStableFunc(ranked, func(left, right rankedPlayer) int {
		switch {
		case left.pnl > right.pnl:
			return -1
		case left.pnl < right.pnl:
			return 1
		default:
			return 0
		}
	})
	items := make([]list.Item, 0, len(ranked))
	for _, player := range ranked {
		items = append(items, player.item)
	}
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#071A14")).
		Background(lipgloss.Color("#FFD866")).
		Padding(0, 1)
	selected := min(max(m.resultIndex, 0), max(len(items)-1, 0))
	return bluffListViewWithTitleStyle("Review", items, selected, width, m.recordListHeight(len(items)), &titleStyle)
}

func (m Model) recordBalanceStrip(width int) string {
	playerCount, tableValue, recordedValue := m.recordBalanceValues()
	deficit := tableValue - recordedValue
	balanceValue := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#071A14")).
		Background(colorGreen).
		Padding(0, 1).
		Render("BALANCED")
	if deficit != 0 {
		balanceValue = lipgloss.NewStyle().Bold(true).Foreground(colorRed).Render(signedCredits(deficit))
	}
	formatName := "—"
	if m.table != nil && m.recordFormatIndex >= 0 && m.recordFormatIndex < len(m.table.Formats) {
		formatName = m.table.Formats[m.recordFormatIndex].Name
	}
	stats := []string{
		workspaceStat("FORMAT", formatName),
		workspaceStat("PLAYERS", fmt.Sprintf("%d", playerCount)),
		workspaceStat("TABLE VALUE", credits(tableValue)),
		workspaceStat("RECORDED VALUE", credits(recordedValue)),
		workspaceStat("DEFICIT", balanceValue),
	}
	if width < 70 {
		content := lipgloss.NewStyle().Width(max(width-4, 1)).Align(lipgloss.Center).
			Render(lipgloss.JoinVertical(lipgloss.Center, stats...))
		return lipgloss.NewStyle().Width(width).Padding(0, 1).
			Border(lipgloss.NormalBorder()).BorderForeground(colorMuted).Render(content)
	}
	columnWidth := max((width-14)/5, 9)
	columns := make([]string, 0, len(stats))
	for _, stat := range stats {
		columns = append(columns, lipgloss.NewStyle().Width(columnWidth).Align(lipgloss.Center).Render(stat))
	}
	content := lipgloss.JoinHorizontal(lipgloss.Top, columns[0], "  ", columns[1], "  ", columns[2], "  ", columns[3], "  ", columns[4])
	return lipgloss.NewStyle().Width(width).Padding(0, 1).
		Border(lipgloss.NormalBorder()).BorderForeground(colorMuted).Render(content)
}

func (m Model) recordBalanceValues() (playerCount, tableValue, recordedValue int) {
	if m.table == nil || m.recordFormatIndex < 0 || m.recordFormatIndex >= len(m.table.Formats) {
		return 0, 0, 0
	}
	entry := m.table.Formats[m.recordFormatIndex].RequiredEntry
	for playerID, entered := range m.recordEntered {
		if !entered {
			continue
		}
		playerCount++
		tableValue += entry
		if total, ok := m.recordPlayerTotal(playerID); ok {
			recordedValue += total
		}
	}
	return playerCount, tableValue, recordedValue
}

func (m Model) recordListHeight(items int) int {
	// The default delegate uses two content lines and one spacer per item.
	// Leave room for the recorder header, action bar, and global help bar.
	desired := max(items*3+5, 8)
	if m.height <= 0 {
		return desired
	}
	return min(max(m.height-15, 8), desired)
}

func (m Model) recordPlayerAllIn(playerID string) bool {
	return m.recordEntered[playerID] && len(m.recordCounts[playerID]) == 0
}

// recordPlayerTotal returns the saved chip total for the current game. The
// second result keeps a zero-value entry distinguishable from an unentered
// player.
func (m Model) recordPlayerTotal(playerID string) (int, bool) {
	if !m.recordEntered[playerID] || m.table == nil || m.recordFormatIndex < 0 || m.recordFormatIndex >= len(m.table.Formats) {
		return 0, false
	}
	counts := m.recordCounts[playerID]
	total := 0
	for _, chip := range m.table.Formats[m.recordFormatIndex].Chips {
		total += chip.Value * counts[chip.ID]
	}
	return total, true
}

func (m Model) recordChipCountView() string {
	format := m.table.Formats[m.recordFormatIndex]
	// Keep the reusable counter field inside the popup's content frame at
	// every terminal width.
	contentWidth := max(popupWidth(m.width, 58)-8, 20)
	formView := ""
	if m.form != nil {
		m.form.WithWidth(contentWidth)
		formView = m.form.View()
	}
	finalValue := 0
	for index, chip := range format.Chips {
		if index >= len(m.recordChipValues) {
			continue
		}
		count, err := strconv.Atoi(strings.TrimSpace(m.recordChipValues[index]))
		if err == nil && count >= 0 {
			finalValue += chip.Value * count
		}
	}
	pnlStyle := standingStyle(finalValue - format.RequiredEntry)
	summaryRows := []string{
		earningsSummaryRow("Final value", credits(finalValue), valueStyle),
		earningsSummaryRow("Buy-in", credits(format.RequiredEntry), mutedStyle),
		earningsSummaryRow("P/L", signedCredits(finalValue-format.RequiredEntry), pnlStyle),
	}
	summary := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Right).Render(strings.Join(summaryRows, "\n"))
	return lipgloss.JoinVertical(lipgloss.Left,
		formView,
		"",
		summary)
}

func earningsSummaryRow(label, amount string, style lipgloss.Style) string {
	labelCell := lipgloss.NewStyle().Width(12).Align(lipgloss.Right).Render(style.Render(label))
	amountCell := lipgloss.NewStyle().Width(12).Align(lipgloss.Right).Render(style.Render(amount))
	return labelCell + "  " + amountCell
}

func (m Model) recordReviewView(width int) string {
	lines := []string{brandStyle.Render("Review game"), mutedStyle.Render(m.recordDetails.date), ""}
	if m.recordPreview == nil {
		return strings.Join(append(lines, mutedStyle.Render("Press enter to check the table balance.")), "\n")
	}
	game := m.recordPreview
	lines = append(lines, valueStyle.Render(fmt.Sprintf("Expected %s  ·  Actual %s", credits(game.ExpectedTableValue), credits(game.ActualTableValue))), "")
	for _, participant := range game.Participants {
		lines = append(lines, fmt.Sprintf("%-18s %8s  %s", truncate(displayParticipantName(participant.PlayerName, m.table.Table.HostUsername), 18), signedCredits(participant.ProfitLoss), standingStyle(participant.EndingStanding).Render(signedCredits(participant.EndingStanding))))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(colorGreen).Render("✓ Balanced and ready to record"))
	return strings.Join(lines, "\n")
}

func chipStyle(color string) lipgloss.Style {
	value := colorMuted
	switch strings.ToLower(color) {
	case "white":
		value = colorCream
	case "black":
		value = lipgloss.Color("#6E6E78")
	case "red":
		value = colorRed
	case "blue":
		value = colorIndigo
	case "green":
		value = colorGreen
	case "yellow":
		value = lipgloss.Color("#F2C94C")
	case "orange":
		value = lipgloss.Color("#F2994A")
	case "gray", "grey":
		value = lipgloss.Color("#8B8B92")
	case "pink":
		value = colorFuchsia
	}
	return lipgloss.NewStyle().Foreground(value)
}

// chipSwatch renders a compact filled rectangle, like a terminal color block.
func chipSwatch(color string) string {
	return chipStyle(color).Render("██")
}

func (m *Model) resetTableCreateForm() {
	m.tableForm = &tableFormValues{}
	tableName := newHuhInput("Table name", "", "saturday-table", &m.tableForm.name, 48, false, publicTableName).
		Prompt("#")
	m.form = newHuhForm(huh.NewGroup(tableName))
	m.resizeForm()
}

func (m *Model) resetFormatCreateForm() {
	m.formatEditIndex = -1
	m.resetFormatForm(nil)
}

func (m *Model) resetFormatEditForm() {
	if m.table == nil || m.formatIndex < 0 || m.formatIndex >= len(m.table.Formats) {
		m.resetFormatCreateForm()
		return
	}
	m.formatEditIndex = m.formatIndex
	format := m.table.Formats[m.formatIndex]
	m.resetFormatForm(&format)
}

func (m *Model) resetFormatForm(existing *api.GameFormat) {
	colors := []string{"white", "black", "green", "blue", "red", "yellow", "orange", "gray", "pink"}
	name, entry := "", ""
	values := make(map[string]string)
	if existing != nil {
		name, entry = existing.Name, strconv.Itoa(existing.RequiredEntry)
		for _, chip := range existing.Chips {
			color := strings.ToLower(strings.TrimSpace(chip.Color))
			if color == "" {
				color = strings.ToLower(strings.TrimSpace(chip.Label))
			}
			if color == "" {
				continue
			}
			if _, present := values[color]; !present {
				values[color] = strconv.Itoa(chip.Value)
				if !containsString(colors, color) {
					colors = append(colors, color)
				}
			}
		}
	}
	m.formatForm = &formatFormValues{name: name, requiredEntry: entry, chips: make([]chipFormValue, len(colors))}
	fields := []huh.Field{
		newHuhInput("Format name", "", "saturday 2k", &m.formatForm.name, 32, false, required("enter a format name")),
		newHuhInput("Total buy-in", "", "2000", &m.formatForm.requiredEntry, 10, false, positiveIntegerText),
	}
	chipSpecs := make([]chipInputSpec, 0, len(colors))
	for index, color := range colors {
		m.formatForm.chips[index].color = color
		m.formatForm.chips[index].value = values[color]
		chipSpecs = append(chipSpecs, chipInputSpec{color: color, placeholder: "value", value: &m.formatForm.chips[index].value})
	}
	fields = append(fields, newHorizontalChipInputs(chipSpecs))
	m.form = newHuhForm(huh.NewGroup(
		fields...,
	))
	m.resizeForm()
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (m *Model) resetPlayerCreateForm() {
	m.playerForm = &playerFormValues{}
	m.form = newHuhForm(huh.NewGroup(
		newHuhInput("Player name", "", "john", &m.playerForm.name, 48, false, required("enter a player name")),
		popupConfirm("Generate invite code", &m.playerForm.generateInvite),
	))
	m.resizeForm()
}

func (m *Model) resetPlayerEditForm() {
	m.playerDeleteConfirm = false
	if m.table == nil || m.playerIndex < 0 || m.playerIndex >= len(m.table.Players) {
		m.form = nil
		return
	}
	player := m.table.Players[m.playerIndex]
	if strings.TrimSpace(player.Username) != "" || !m.table.CanManage {
		m.playerForm, m.form = nil, nil
		return
	}
	m.playerForm = &playerFormValues{name: player.Name}
	fields := []huh.Field{
		newHuhInput("Player name", "", player.Name, &m.playerForm.name, 48, false, required("enter a player name")),
	}
	if m.playerInviteCodeForCurrentPlayer() == "" {
		fields = append(fields, popupConfirm("Generate invite code", &m.playerForm.generateInvite))
	}
	m.form = newHuhForm(huh.NewGroup(fields...))
	m.resizeForm()
}

func (m *Model) openRecordMetadataPopup() tea.Cmd {
	if m.recordDetails == nil {
		m.recordDetails = &recordDetailsValues{date: time.Now().Format("2006-01-02")}
	}
	m.recordMetadataDraft = &recordDetailsValues{date: m.recordDetails.date, remarks: m.recordDetails.remarks}
	m.recordPopup, m.err = recordMetadataPopup, nil
	m.form = newHuhForm(huh.NewGroup(
		newHuhInput("Game date", "", m.recordMetadataDraft.date, &m.recordMetadataDraft.date, 10, false, isoDateText),
		newHuhText("Note", "", "optional note", &m.recordMetadataDraft.remarks, 4, 500, nil),
	))
	m.resizeForm()
	return m.form.Init()
}

func (m *Model) resetRecordChipForm() {
	format := m.table.Formats[m.recordFormatIndex]
	player := m.table.Players[m.recordPlayerIndex]
	counts := m.recordCounts[player.ID]
	if counts == nil {
		counts = map[string]int{}
		m.recordCounts[player.ID] = counts
	}
	// An empty saved count map represents a player who was marked all in. A
	// fresh player still gets the normal denomination form.
	m.recordAllIn = m.recordEntered[player.ID] && len(counts) == 0
	values := make([]string, 0, len(format.Chips))
	for _, chip := range format.Chips {
		value := strconv.Itoa(counts[chip.ID])
		values = append(values, value)
	}
	m.recordChipValues = values
	m.rebuildRecordChipForm()
}

func (m *Model) rebuildRecordChipForm() {
	format := m.table.Formats[m.recordFormatIndex]
	allIn := m.recordAllIn
	m.recordAllInValue = &allIn
	fields := []huh.Field{popupStackedConfirm("All in?", m.recordAllInValue)}
	if !m.recordAllIn {
		fields = append(fields, newVerticalChipCounters(format.Chips, &m.recordChipValues))
	}
	m.form = newHuhForm(huh.NewGroup(fields...))
	m.resizeForm()
}

func positiveIntegerText(value string) error {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number <= 0 {
		return errors.New("enter a positive whole number")
	}
	return nil
}

func publicTableName(value string) error {
	slug := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "#")
	if slug == "" || !hasOnlyTableNameCharacters(slug) {
		return errors.New("use letters, numbers, and hyphens")
	}
	return nil
}

func hasOnlyTableNameCharacters(value string) bool {
	hasAlphaNumeric := false
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') {
			hasAlphaNumeric = true
			continue
		}
		if character != '-' {
			return false
		}
	}
	return hasAlphaNumeric
}

func canonicalTableName(value string) string {
	slug := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "#")
	return "#" + slug
}

func nonNegativeIntegerText(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("enter zero or a positive whole number")
	}
	if _, err := strconv.Atoi(strings.TrimSpace(value)); err != nil || strings.HasPrefix(strings.TrimSpace(value), "-") {
		return errors.New("enter zero or a positive whole number")
	}
	return nil
}

func isoDateText(value string) error {
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
		return errors.New("use YYYY-MM-DD")
	}
	return nil
}

func chipSpec(value string) error {
	if len(parseChipSpec(value)) == 0 {
		return errors.New("use values like white=10, black=100")
	}
	return nil
}

func parseChipSpec(value string) []api.ChipDenomination {
	var chips []api.ChipDenomination
	seen := map[string]bool{}
	for index, raw := range strings.Split(value, ",") {
		parts := strings.SplitN(strings.TrimSpace(raw), "=", 2)
		if len(parts) != 2 {
			continue
		}
		label := strings.TrimSpace(parts[0])
		amount, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		key := strings.ToLower(label)
		if label == "" || err != nil || amount <= 0 || seen[key] {
			continue
		}
		seen[key] = true
		chips = append(chips, api.ChipDenomination{Label: label, Color: key, Value: amount, Position: index})
	}
	return chips
}

func parseChipRows(rows []chipFormValue) ([]api.ChipDenomination, error) {
	chips := make([]api.ChipDenomination, 0, len(rows))
	seen := map[string]bool{}
	for index, row := range rows {
		color := strings.ToLower(strings.TrimSpace(row.color))
		value := strings.TrimSpace(row.value)
		if value == "" {
			continue
		}
		if color == "" {
			return nil, errors.New("complete each chip denomination")
		}
		amount, err := strconv.Atoi(value)
		if err != nil || amount <= 0 {
			return nil, errors.New("chip denominations must be positive whole numbers")
		}
		if seen[color] {
			return nil, errors.New("each chip color can appear only once")
		}
		seen[color] = true
		chips = append(chips, api.ChipDenomination{Label: color, Color: color, Value: amount, Position: index})
	}
	if len(chips) == 0 {
		return nil, errors.New("add at least one chip denomination")
	}
	return chips, nil
}

func (m Model) isTableFormScreen() bool {
	return m.screen == tableCreateScreen || m.screen == formatCreateScreen || m.screen == playerCreateScreen ||
		(m.screen == playerDetailScreen && m.form != nil) ||
		(m.screen == recordGameScreen && (m.recordPopup != recordPopupNone || m.recordPhase == recordChipCountsPhase))
}

func (m Model) tablesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		tables, err := m.api.Tables(ctx, m.token)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return tablesLoadedMsg{tables: tables}
	}
}

func (m Model) tableCmd(tableID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		table, err := m.api.Table(ctx, m.token, tableID)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return tableLoadedMsg{table: table}
	}
}

func (m Model) createTableCmd() tea.Cmd {
	name := canonicalTableName(m.tableForm.name)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		table, err := m.api.CreateTable(ctx, m.token, name)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return tableCreatedMsg{table: table}
	}
}

func (m Model) createPlayerCmd() tea.Cmd {
	tableID, name, generateInvite := m.table.Table.ID, strings.ToLower(strings.TrimSpace(m.playerForm.name)), m.playerForm.generateInvite
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		player, err := m.api.CreateTablePlayer(ctx, m.token, tableID, name)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		var inviteCode string
		if generateInvite {
			// TODO: replace this account invite with a player-bound invite once
			// the API exposes the player claim endpoint.
			invitation, inviteErr := m.api.CreateInvitation(ctx, m.token)
			if inviteErr != nil {
				return operationFailedMsg{err: inviteErr}
			}
			inviteCode = invitation.Code
		}
		return tablePlayerCreatedMsg{player: player, inviteCode: inviteCode}
	}
}

func (m Model) updatePlayerCmd() tea.Cmd {
	tableID := m.table.Table.ID
	playerID := m.table.Players[m.playerIndex].ID
	name := strings.ToLower(strings.TrimSpace(m.playerForm.name))
	generateInvite := m.playerForm.generateInvite
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		player, err := m.api.UpdateTablePlayer(ctx, m.token, tableID, playerID, name)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		var inviteCode string
		if generateInvite {
			invitation, inviteErr := m.api.CreateInvitation(ctx, m.token)
			if inviteErr != nil {
				return operationFailedMsg{err: inviteErr}
			}
			inviteCode = invitation.Code
		}
		return tablePlayerUpdatedMsg{player: player, inviteCode: inviteCode}
	}
}

func (m Model) createPlayerInviteCmd() tea.Cmd {
	if m.table == nil || m.playerIndex < 0 || m.playerIndex >= len(m.table.Players) {
		return nil
	}
	playerID := m.table.Players[m.playerIndex].ID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		invitation, err := m.api.CreateInvitation(ctx, m.token)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return playerInviteCreatedMsg{playerID: playerID, code: invitation.Code}
	}
}

func (m Model) playerHasEntries() bool {
	if m.table == nil || m.playerIndex < 0 || m.playerIndex >= len(m.table.Players) {
		return false
	}
	playerID := m.table.Players[m.playerIndex].ID
	for _, game := range m.table.Games {
		for _, participant := range game.Participants {
			if participant.PlayerID == playerID {
				return true
			}
		}
	}
	return false
}

func (m Model) deletePlayerCmd() tea.Cmd {
	tableID := m.table.Table.ID
	playerID := m.table.Players[m.playerIndex].ID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		if err := m.api.DeleteTablePlayer(ctx, m.token, tableID, playerID); err != nil {
			return operationFailedMsg{err: err}
		}
		return tablePlayerRemovedMsg{playerID: playerID}
	}
}

func (m Model) disablePlayerCmd() tea.Cmd {
	tableID := m.table.Table.ID
	playerID := m.table.Players[m.playerIndex].ID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		if err := m.api.DisableTablePlayer(ctx, m.token, tableID, playerID); err != nil {
			return operationFailedMsg{err: err}
		}
		return tablePlayerRemovedMsg{playerID: playerID, disabled: true}
	}
}

func (m Model) createFormatCmd() tea.Cmd {
	tableID, form := m.table.Table.ID, *m.formatForm
	entry, _ := strconv.Atoi(strings.TrimSpace(form.requiredEntry))
	chips, _ := parseChipRows(form.chips)
	name := strings.ToLower(strings.TrimSpace(form.name))
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		format, err := m.api.CreateGameFormat(ctx, m.token, tableID, name, entry, chips)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return tableFormatCreatedMsg{format: format}
	}
}

func (m Model) updateFormatCmd() tea.Cmd {
	tableID := m.table.Table.ID
	formatID := m.table.Formats[m.formatEditIndex].ID
	index := m.formatEditIndex
	form := *m.formatForm
	entry, _ := strconv.Atoi(strings.TrimSpace(form.requiredEntry))
	chips, _ := parseChipRows(form.chips)
	name := strings.ToLower(strings.TrimSpace(form.name))
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		format, err := m.api.UpdateGameFormat(ctx, m.token, tableID, formatID, name, entry, chips)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return tableFormatUpdatedMsg{index: index, format: format}
	}
}

func (m Model) recordParticipants() []api.GameParticipantInput {
	format := m.table.Formats[m.recordFormatIndex]
	participants := make([]api.GameParticipantInput, 0, len(m.recordEntered))
	for _, player := range m.table.Players {
		if !m.recordEntered[player.ID] {
			continue
		}
		counts := m.recordCounts[player.ID]
		input := api.GameParticipantInput{PlayerID: player.ID, ChipCounts: make([]api.ChipCountInput, 0, len(format.Chips))}
		for _, chip := range format.Chips {
			input.ChipCounts = append(input.ChipCounts, api.ChipCountInput{DenominationID: chip.ID, Count: counts[chip.ID]})
		}
		participants = append(participants, input)
	}
	return participants
}

func (m Model) previewRecordGameCmd() tea.Cmd {
	tableID := m.table.Table.ID
	formatID := m.table.Formats[m.recordFormatIndex].ID
	date, remarks, participants := m.recordDetails.date, m.recordDetails.remarks, m.recordParticipants()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		var game api.TableGame
		var err error
		if m.recordEditGameID != "" {
			game, err = m.api.PreviewTableGameEdit(ctx, m.token, tableID, m.recordEditGameID, formatID, date, remarks, participants)
		} else {
			game, err = m.api.PreviewTableGame(ctx, m.token, tableID, formatID, date, remarks, participants)
		}
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return tableGamePreviewedMsg{game: game}
	}
}

func (m Model) recordGameCmd() tea.Cmd {
	tableID := m.table.Table.ID
	formatID := m.table.Formats[m.recordFormatIndex].ID
	date, remarks, participants := m.recordDetails.date, m.recordDetails.remarks, m.recordParticipants()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		var table api.TableDetail
		var err error
		if m.recordEditGameID != "" {
			table, err = m.api.UpdateTableGame(ctx, m.token, tableID, m.recordEditGameID, formatID, date, remarks, m.recordEditVersion, participants)
		} else {
			table, err = m.api.RecordTableGame(ctx, m.token, tableID, formatID, date, remarks, participants)
		}
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return tableGameRecordedMsg{table: table}
	}
}
