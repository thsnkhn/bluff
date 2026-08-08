package ui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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
type tableGamePreviewedMsg struct{ game api.TableGame }
type tableGameRecordedMsg struct{ table api.TableDetail }

func (m Model) isTableScreen() bool {
	switch m.screen {
	case tablesScreen, tableDetailScreen, formatsScreen, formatDetailScreen, playersScreen,
		gamesScreen, gameDetailScreen, tableCreateScreen, formatCreateScreen, playerCreateScreen, playerDetailScreen, recordGameScreen:
		return true
	default:
		return false
	}
}

func (m Model) isTableInteractiveScreen() bool { return m.isTableScreen() && !m.loading }

func (m Model) tableParentScreen() screen {
	if m.screen == playerCreateScreen && m.recordQuickAdd {
		return recordGameScreen
	}
	switch m.screen {
	case formatCreateScreen, formatsScreen, formatDetailScreen, playerCreateScreen, playerDetailScreen, playersScreen,
		gamesScreen, gameDetailScreen:
		return tableDetailScreen
	case recordGameScreen:
		return tableDetailScreen
	default:
		return tablesScreen
	}
}

func (m Model) updateTableKey(key string) (tea.Model, tea.Cmd, bool) {
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
		case "r":
			m.loading, m.status, m.err = true, "Refreshing table", nil
			return m, tea.Batch(m.spinner.Tick, m.tableCmd(m.table.Table.ID)), true
		case "c":
			if m.table.CanManage {
				m.startRecordGame()
				return m, m.form.Init(), true
			}
		case "esc", "backspace":
			m.screen, m.err = tablesScreen, nil
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
			if len(m.table.Formats) > 0 {
				m.screen = formatDetailScreen
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
	case formatDetailScreen:
		if key == "esc" || key == "backspace" {
			m.screen, m.err = formatsScreen, nil
			return m, nil, true
		}
	case gamesScreen:
		switch key {
		case "up", "k":
			moveVisible(&m.gameIndex, m.visibleGameIndices(), -1)
			return m, nil, true
		case "down", "j":
			moveVisible(&m.gameIndex, m.visibleGameIndices(), 1)
			return m, nil, true
		case "enter", " ":
			if len(m.visibleGameIndices()) > 0 {
				m.screen = gameDetailScreen
			}
			return m, nil, true
		case "c":
			if m.table.CanManage {
				m.startRecordGame()
				return m, m.form.Init(), true
			}
		case "esc", "backspace":
			m.screen, m.err = tableDetailScreen, nil
			return m, nil, true
		}
	case gameDetailScreen:
		if key == "esc" || key == "backspace" {
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
		if key == "enter" && m.form != nil && m.table.CanManage && m.playerCanEdit() {
			updated, cmd := m.handleTableFormCompleted()
			return updated, cmd, true
		}
		if key == "c" && m.table.CanManage && m.playerInviteCodeForCurrentPlayer() == "" {
			m.loading, m.status, m.err = true, "Creating invite code", nil
			return m, tea.Batch(m.spinner.Tick, m.createPlayerInviteCmd()), true
		}
		if key == "d" && m.table.CanManage && !m.playerHasEntries() {
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
		switch m.recordPhase {
		case recordDetailsPhase, recordChipCountsPhase:
			if key == "esc" {
				m.screen, m.err = tableDetailScreen, nil
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
				m.recordSelected = map[string]bool{}
				return m, nil, true
			case "esc", "backspace":
				m.screen, m.err = tableDetailScreen, nil
				return m, nil, true
			}
		case recordPlayersPhase:
			switch key {
			case "up", "k":
				m.playerIndex = max(m.playerIndex-1, 0)
				return m, nil, true
			case "down", "j":
				m.playerIndex = min(m.playerIndex+1, max(len(m.table.Players)-1, 0))
				return m, nil, true
			case "space":
				if len(m.table.Players) > 0 {
					player := m.table.Players[m.playerIndex]
					m.recordSelected[player.ID] = !m.recordSelected[player.ID]
				}
				return m, nil, true
			case "enter", "e":
				// Enter on a selected player with saved chips edits that player's
				// earnings. Otherwise start the first unfinished player.
				if m.recordSelectedPlayerIsEntered() {
					m.recordPlayerIndex = m.playerIndex
					m.recordPhase = recordChipCountsPhase
					m.resetRecordChipForm()
					return m, m.form.Init(), true
				}
				if len(m.selectedPlayerIndices()) < 2 {
					m.err = errors.New("select at least two players")
					return m, nil, true
				}
				indices := m.selectedPlayerIndices()
				m.recordPlayerIndex = m.firstUnenteredRecordPlayer(indices)
				m.recordPhase = recordChipCountsPhase
				m.resetRecordChipForm()
				return m, m.form.Init(), true
			case "r":
				if m.allSelectedRecordingsEntered() {
					m.loading, m.status, m.err = true, "Checking table balance", nil
					return m, tea.Batch(m.spinner.Tick, m.previewRecordGameCmd()), true
				}
				return m, nil, true
			case "d":
				if m.table.CanManage && m.recordSelectedPlayerIsEntered() {
					player := m.table.Players[m.playerIndex]
					delete(m.recordCounts, player.ID)
					delete(m.recordEntered, player.ID)
					m.recordPreview, m.err = nil, nil
				}
				return m, nil, true
			case "c":
				if m.table.CanManage {
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
			if key == "esc" || key == "backspace" {
				m.screen, m.err = tableDetailScreen, nil
				return m, nil, true
			}
			if key == "enter" || key == "r" {
				m.loading, m.err = true, nil
				if m.recordPreview == nil {
					m.status = "Checking table balance"
					return m, tea.Batch(m.spinner.Tick, m.previewRecordGameCmd()), true
				}
				m.status = "Recording game"
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
			m.resetFormatCreateForm()
			return m, m.form.Init()
		} else {
			_ = chips
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
		if m.recordPhase == recordDetailsPhase {
			m.form = nil
			m.recordPhase, m.err = recordFormatPhase, nil
			return m, nil
		}
		if m.recordPhase == recordChipCountsPhase {
			player := m.table.Players[m.recordPlayerIndex]
			counts := map[string]int{}
			format := m.table.Formats[m.recordFormatIndex]
			for index, chip := range format.Chips {
				value, err := strconv.Atoi(strings.TrimSpace(m.recordChipValues[index]))
				if err != nil || value < 0 {
					m.err = errors.New("chip counts must be whole numbers")
					return m, nil
				}
				counts[chip.ID] = value
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

func (m Model) selectedPlayerIndices() []int {
	indices := make([]int, 0, len(m.recordSelected))
	for index, player := range m.table.Players {
		if m.recordSelected[player.ID] {
			indices = append(indices, index)
		}
	}
	return indices
}

func (m Model) recordSelectedPlayerIsEntered() bool {
	if m.playerIndex < 0 || m.playerIndex >= len(m.table.Players) {
		return false
	}
	player := m.table.Players[m.playerIndex]
	return m.recordSelected[player.ID] && m.recordEntered[player.ID]
}

func (m Model) firstUnenteredRecordPlayer(indices []int) int {
	for _, index := range indices {
		if !m.recordEntered[m.table.Players[index].ID] {
			return index
		}
	}
	return indices[0]
}

func (m Model) allSelectedRecordingsEntered() bool {
	indices := m.selectedPlayerIndices()
	if len(indices) < 2 {
		return false
	}
	for _, index := range indices {
		if !m.recordEntered[m.table.Players[index].ID] {
			return false
		}
	}
	return true
}

func (m *Model) startRecordGame() {
	m.screen, m.recordPhase, m.recordPreview, m.err = recordGameScreen, recordDetailsPhase, nil, nil
	m.recordSelected, m.recordCounts = map[string]bool{}, map[string]map[string]int{}
	m.recordEntered = map[string]bool{}
	m.recordQuickAdd, m.recordQuickAddID = false, ""
	m.resetRecordDetailsForm()
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
				return m, m.form.Init()
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
				m.screen = formatDetailScreen
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
				m.screen = gameDetailScreen
			}
			return m, nil
		}
		if msg.action == "search" && msg.activate {
			m.searchActive, m.searchQuery = true, ""
		}
		if msg.action == "record" && msg.activate && m.table.CanManage {
			m.startRecordGame()
			return m, m.form.Init()
		}
		if msg.action == "back" && msg.activate {
			m.screen = tableDetailScreen
		}
	case formatDetailScreen, gameDetailScreen:
		if msg.action == "search" && msg.activate {
			m.searchActive, m.searchQuery = true, ""
		}
	case recordGameScreen:
		if msg.action != "" && msg.activate {
			switch msg.action {
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
				if m.recordPhase == recordPlayersPhase && m.recordSelectedPlayerIsEntered() {
					player := m.table.Players[m.playerIndex]
					delete(m.recordCounts, player.ID)
					delete(m.recordEntered, player.ID)
					m.recordPreview, m.err = nil, nil
				}
				return m, nil
			case "edit":
				if m.recordPhase == recordPlayersPhase && m.recordSelectedPlayerIsEntered() {
					m.recordPlayerIndex = m.playerIndex
					m.recordPhase = recordChipCountsPhase
					m.resetRecordChipForm()
					return m, m.form.Init()
				}
				return m, nil
			case "review":
				if m.recordPhase == recordPlayersPhase && m.allSelectedRecordingsEntered() {
					m.loading, m.status, m.err = true, "Checking table balance", nil
					return m, tea.Batch(m.spinner.Tick, m.previewRecordGameCmd())
				}
			case "record":
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
				player := m.table.Players[msg.index]
				m.recordSelected[player.ID] = !m.recordSelected[player.ID]
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
			regions = append(regions, hitRegion{x0: x, x1: x + width, y0: listY + 1 + row, y1: listY + 1 + row, value: fmt.Sprintf("table:%d", index)})
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
			regions = append(regions, hitRegion{x0: x, x1: x + width, y0: listY + 5 + row, y1: listY + 5 + row, value: fmt.Sprintf("format:%d", index)})
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
			regions = append(regions, hitRegion{x0: x, x1: x + width, y0: listY + 5 + row, y1: listY + 5 + row, value: fmt.Sprintf("player:%d", index)})
		}
	case formatDetailScreen:
		if m.table == nil || m.formatIndex < 0 || m.formatIndex >= len(m.table.Formats) {
			return nil
		}
		items = []actionBarItem{{key: "/", label: "Search", action: "search"}, {key: "esc", label: "Back", action: "back"}}
		actionY = y + lipgloss.Height(pageHeader(width, m.table.Table.Name, "formats", m.table.Formats[m.formatIndex].Name)) + 1
		regions = actionBarHitRegions(x, actionY, items)
	case gameDetailScreen:
		if m.table == nil || m.gameIndex < 0 || m.gameIndex >= len(m.table.Games) {
			return nil
		}
		items = []actionBarItem{{key: "/", label: "Search", action: "search"}, {key: "esc", label: "Back", action: "back"}}
		actionY = y + lipgloss.Height(pageHeader(width, m.table.Table.Name, "games", m.table.Games[m.gameIndex].Date)) + 1
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
			regions = append(regions, hitRegion{x0: x, x1: x + width, y0: listY + row + 5, y1: listY + row + 5, value: fmt.Sprintf("game:%d", index)})
		}
	case recordGameScreen:
		if m.table == nil {
			return nil
		}
		actionY := y + lipgloss.Height(pageHeader(width, m.table.Table.Name, "record")) + 1
		regions = append(regions, actionBarHitRegions(x, actionY, m.recordActionItems())...)
		baseY := actionY + 2
		if m.recordPhase == recordFormatPhase {
			for index := range m.table.Formats {
				regions = append(regions, hitRegion{x0: x, x1: x + width, y0: baseY + 2 + index, y1: baseY + 2 + index, value: fmt.Sprintf("format:%d", index)})
			}
		}
		if m.recordPhase == recordPlayersPhase {
			for index := range m.table.Players {
				regions = append(regions, hitRegion{x0: x, x1: x + width, y0: baseY + 3 + index, y1: baseY + 3 + index, value: fmt.Sprintf("player:%d", index)})
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
	parts := []string{pageHeader(width, "tables"), "", searchActionBar(items, m.tablesActionHover, m.searchActive, m.searchQuery), "", m.tableSummaryList(width)}
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
		return lipgloss.JoinVertical(lipgloss.Left, valueStyle.Render("No tables yet"), mutedStyle.Render("Create a table to start a private ledger."))
	}
	if len(m.visibleTableIndices()) == 0 {
		return mutedStyle.Render("No tables match the search.")
	}
	columns := tableSummaryColumns(width)
	lines := []string{mutedStyle.Render(tableSummaryGridLine(columns, "TABLE", "HOST", "PLAYERS", "GAMES", "LAST GAME"))}
	for _, index := range m.visibleTableIndices() {
		table := m.tables[index]
		style := valueStyle
		marker := "  "
		if index == m.tableIndex {
			style = lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia)
			marker = "› "
		}
		lastGame := "—"
		if table.LastGameDate != nil {
			lastGame = *table.LastGameDate
		}
		host := "@" + truncate(table.HostUsername, 11) + " " + lipgloss.NewStyle().Foreground(colorFuchsia).Render("♛")
		line := tableSummaryGridLine(columns,
			marker+truncate(table.Name, max(columns[0]-2, 1)),
			host,
			strconv.Itoa(table.PlayerCount),
			strconv.Itoa(table.GameCount),
			lastGame,
		)
		lines = append(lines, style.Render(line))
	}
	return strings.Join(lines, "\n")
}

// tableSummaryColumns spreads the table index across the full terminal width.
// Keep the proportions stable while allowing the last column to absorb rounding.
func tableSummaryColumns(width int) []int {
	content := max(width-8, 20) // four two-space gutters
	tableWidth := content * 30 / 100
	hostWidth := content * 24 / 100
	playersWidth := content * 14 / 100
	gamesWidth := content * 14 / 100
	lastWidth := content - tableWidth - hostWidth - playersWidth - gamesWidth
	return []int{tableWidth, hostWidth, playersWidth, gamesWidth, lastWidth}
}

func tableSummaryGridLine(columns []int, values ...string) string {
	if len(columns) != len(values) {
		return strings.Join(values, "  ")
	}
	cells := make([]string, 0, len(values))
	for index, value := range values {
		cellWidth := max(columns[index], 1)
		cells = append(cells, lipgloss.NewStyle().Width(cellWidth).Render(truncate(value, cellWidth)))
	}
	return strings.Join(cells, "  ")
}

func weightedGridColumns(width int, weights ...int) []int {
	if len(weights) == 0 {
		return nil
	}
	gutters := (len(weights) - 1) * 2
	content := max(width-gutters, len(weights))
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
	content := m.tableOverviewContent(workspaceContentWidth(metrics))
	if m.loading {
		content = m.spinner.View() + "  " + valueStyle.Render(m.status)
	}
	if m.notice != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", lipgloss.NewStyle().Foreground(colorGreen).Render("✓ "+m.notice))
	}
	return m.tableWorkspace(tableOverviewSection, items, content, tableDetailFooter())
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
		return "↑↓ move   enter inspect   c create"
	}
	return "↑↓ move   enter inspect"
}

func (m Model) formatList(width int) string {
	if len(m.table.Formats) == 0 {
		return workspaceEmptyState(width, "No game formats yet", "The table host can add the first format.")
	}
	if len(m.visibleFormatIndices()) == 0 {
		return lipgloss.NewStyle().Width(width).Render(mutedStyle.Render("No formats match the search."))
	}
	columns := weightedGridColumns(width, 52, 16, 32)
	lines := []string{mutedStyle.Render(tableSummaryGridLine(columns, "FORMAT", "ENTRY", "CHIPS"))}
	for _, index := range m.visibleFormatIndices() {
		format := m.table.Formats[index]
		style := valueStyle
		marker := "  "
		if index == m.formatIndex {
			style = lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia)
			marker = "› "
		}
		chipNames := make([]string, 0, len(format.Chips))
		for _, chip := range format.Chips {
			chipNames = append(chipNames, fmt.Sprintf("%s %s %d", chipSwatch(chip.Color), chip.Label, chip.Value))
		}
		line := tableSummaryGridLine(columns,
			marker+truncate(format.Name, max(columns[0]-2, 1)),
			credits(format.RequiredEntry),
			strings.Join(chipNames, ", "),
		)
		lines = append(lines, style.Render(line))
	}
	return strings.Join(lines, "\n")
}

func (m Model) formatDetailView() string {
	width := max(m.width-4, 44)
	if m.table == nil || m.formatIndex < 0 || m.formatIndex >= len(m.table.Formats) {
		return m.pageView(pageHeader(width, "formats"), "")
	}
	format := m.table.Formats[m.formatIndex]
	lines := []string{pageHeader(width, m.table.Table.Name, "formats", format.Name), "", searchActionBar([]actionBarItem{{key: "/", label: "Search", action: "search"}, {key: "esc", label: "Back", action: "back"}}, "", m.searchActive, m.searchQuery), "", valueStyle.Render("Required entry  " + credits(format.RequiredEntry)), "", sectionHeading("Chip denominations", width)}
	for _, chip := range format.Chips {
		if !searchMatches(m.searchQuery, chip.Label+" "+chip.Color+" "+credits(chip.Value)) {
			continue
		}
		lines = append(lines, chipSwatch(chip.Color)+" "+valueStyle.Render(chip.Label)+"  "+valueStyle.Render(credits(chip.Value)))
	}
	return m.pageView(strings.Join(lines, "\n"), "")
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
		return lipgloss.NewStyle().Width(width).Render(mutedStyle.Render("No games match the search."))
	}
	columns := weightedGridColumns(width, 18, 42, 15, 25)
	lines := []string{mutedStyle.Render(tableSummaryGridLine(columns, "DATE", "FORMAT", "PLAYERS", "STATUS"))}
	visible := m.visibleGameIndices()
	for row := len(visible) - 1; row >= 0; row-- {
		index := visible[row]
		game := m.table.Games[index]
		style := valueStyle
		marker := "  "
		if index == m.gameIndex {
			style = lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia)
			marker = "› "
		}
		line := tableSummaryGridLine(columns,
			marker+truncate(game.Date, max(columns[0]-2, 1)),
			truncate(game.Format.Name, max(columns[1], 1)),
			strconv.Itoa(len(game.Participants)),
			statusBadge(game.Status),
		)
		lines = append(lines, style.Render(line))
	}
	return strings.Join(lines, "\n")
}

func (m Model) gameDetailView() string {
	width := max(m.width-4, 44)
	if m.table == nil || m.gameIndex < 0 || m.gameIndex >= len(m.table.Games) {
		return m.pageView(pageHeader(width, "tables", "games"), "")
	}
	game := m.table.Games[m.gameIndex]
	lines := []string{
		pageHeader(width, m.table.Table.Name, "games", game.Date),
		"",
		searchActionBar([]actionBarItem{{key: "/", label: "Search", action: "search"}, {key: "esc", label: "Back", action: "back"}}, "", m.searchActive, m.searchQuery),
		"",
		valueStyle.Render(game.Format.Name),
		mutedStyle.Render(fmt.Sprintf("%s  ·  required entry %s  ·  %s", game.Date, credits(game.Format.RequiredEntry), statusBadge(game.Status))),
		"",
		sectionHeading("Chip snapshot", width),
	}
	for _, chip := range game.Format.Chips {
		lines = append(lines, "  "+chipSwatch(chip.Color)+" "+valueStyle.Render(chip.Label)+"  "+valueStyle.Render(credits(chip.Value)))
	}
	lines = append(lines, "", sectionHeading("Results", width))
	for _, participant := range game.Participants {
		if !searchMatches(m.searchQuery, participant.PlayerName) {
			continue
		}
		lines = append(lines, valueStyle.Render(truncate(displayParticipantName(participant.PlayerName, m.table.Table.HostUsername), 22))+"  "+
			standingStyle(participant.ProfitLoss).Render(signedCredits(participant.ProfitLoss))+"  "+
			mutedStyle.Render("ending ")+standingStyle(participant.EndingStanding).Render(signedCredits(participant.EndingStanding)))
		for _, chip := range participant.ChipCounts {
			if chip.Count > 0 {
				lines = append(lines, mutedStyle.Render(fmt.Sprintf("    %s × %d = %s", chip.Label, chip.Count, credits(chip.Value*chip.Count))))
			}
		}
	}
	lines = append(lines, "", valueStyle.Render(fmt.Sprintf("Expected %s  ·  Actual %s", credits(game.ExpectedTableValue), credits(game.ActualTableValue))))
	if game.Remarks != "" {
		lines = append(lines, mutedStyle.Render("Note: "+game.Remarks))
	}
	return m.pageView(strings.Join(lines, "\n"), "")
}

func (m Model) playerList(width int) string {
	if len(m.table.Players) == 0 {
		return workspaceEmptyState(width, "No players yet", "Add a player before recording a game.")
	}
	if len(m.visiblePlayerIndices()) == 0 {
		return lipgloss.NewStyle().Width(width).Render(mutedStyle.Render("No players match the search."))
	}
	columns := weightedGridColumns(width, 3, 1)
	lines := []string{mutedStyle.Render(tableSummaryGridLine(columns, "PLAYER", "STANDING"))}
	for _, index := range m.visiblePlayerIndices() {
		player := m.table.Players[index]
		marker := "  "
		style := valueStyle
		if index == m.playerIndex {
			marker = "› "
			style = lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia)
		}
		line := tableSummaryGridLine(columns,
			marker+truncate(displayTablePlayerName(player, m.table.Table.HostUsername), max(columns[0]-2, 1)),
			signedCreditNumber(player.Standing),
		)
		lines = append(lines, style.Render(line))
	}
	return strings.Join(lines, "\n")
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
	return m.formPopupSizedWithActions(background.formatsView(), "Create game format", m.popupFormView(54), "tab next   enter create   esc close", 54, []actionBarItem{
		{key: "enter", label: "Create", action: "submit", accent: true},
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

	details := []string{
		mutedStyle.Render("Standing"),
		standingStyle(player.Standing).Render(signedCredits(player.Standing)),
	}
	if inviteCode != "" {
		details = append(details, "", mutedStyle.Render("Invite code"), brandStyle.Render(inviteCode))
	}
	if m.playerHasEntries() {
		details = append(details, "", mutedStyle.Render("Game entries exist; disable this player to keep the history."))
	} else {
		details = append(details, "", mutedStyle.Render("No game entries yet; this player can be deleted."))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, details...)
	if m.form != nil && m.playerCanEdit() {
		body = lipgloss.JoinVertical(lipgloss.Left, m.form.View(), "", body)
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
			popupActions = append([]actionBarItem{{key: "d", label: "Delete", action: "delete", accent: true}}, popupActions...)
			footer = "d delete   " + footer
		}
	}
	return m.formPopupWithActions(background.playersView(), "Player", body, footer, popupActions)
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
	width := popupWidth(m.width, maxWidth)
	if len(actions) == 0 {
		actions = []actionBarItem{{key: "esc", label: "Close", action: "close"}}
	}
	if m.err != nil {
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	contentWidth := max(width-8, 20)
	popupFooter := popupActionFooter(actions, contentWidth)
	popup := lipgloss.NewStyle().Width(width).Padding(1, 3).
		Border(lipgloss.RoundedBorder()).BorderForeground(colorIndigo).
		Align(lipgloss.Left).
		Render(lipgloss.JoinVertical(lipgloss.Left, popupHeader(title, contentWidth), "", content, "", popupFooter))
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
	header := pageHeader(width, m.table.Table.Name, "record")
	if m.loading {
		return m.pageView(lipgloss.JoinVertical(lipgloss.Left, header, "", m.spinner.View()+"  "+valueStyle.Render(m.status)), "please wait")
	}
	var body string
	switch m.recordPhase {
	case recordDetailsPhase:
		body = lipgloss.JoinVertical(lipgloss.Center, brandStyle.Render("Game details"), "", m.form.View())
	case recordFormatPhase:
		body = m.recordFormatList(width)
	case recordPlayersPhase:
		body = m.recordPlayerList(width)
	case recordChipCountsPhase:
		background := m
		background.recordPhase = recordPlayersPhase
		background.form = nil
		background.err = nil
		return m.formPopupSizedWithActions(background.recordGameView(), "Player earnings", m.recordChipCountView(), recordFooter(m.recordPhase, m.recordPreview != nil), 58, []actionBarItem{
			{key: "enter", label: "Save", action: "submit", accent: true},
			{key: "esc", label: "Close", action: "close"},
		})
	case recordReviewPhase:
		body = m.recordReviewView(width)
	}
	if m.err != nil {
		body = lipgloss.JoinVertical(lipgloss.Center, body, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	actions := actionBar(m.recordActionItems(), "")
	return m.pageView(lipgloss.JoinVertical(lipgloss.Left, header, "", actions, "", body), recordFooter(m.recordPhase, m.recordPreview != nil))
}

func (m Model) recordActionItems() []actionBarItem {
	items := []actionBarItem{{key: "esc", label: "Back", action: "back"}}
	if m.recordPhase == recordPlayersPhase && m.table.CanManage {
		items = append([]actionBarItem{{key: "c", label: "Add player", action: "create", accent: true}}, items...)
		if m.recordSelectedPlayerIsEntered() {
			items = append([]actionBarItem{{key: "e", label: "Edit earning", action: "edit"}}, items...)
			items = append([]actionBarItem{{key: "d", label: "Clear earning", action: "delete"}}, items...)
		}
		if m.allSelectedRecordingsEntered() {
			items = append([]actionBarItem{{key: "r", label: "Review game", action: "review", accent: true}}, items...)
		}
	}
	if m.recordPhase == recordReviewPhase && m.table.CanManage {
		items = append([]actionBarItem{{key: "r", label: "Record game", action: "record", accent: true}}, items...)
	}
	return items
}

func recordFooter(phase recordPhase, previewed bool) string {
	switch phase {
	case recordDetailsPhase:
		return "tab next   enter continue"
	case recordFormatPhase:
		return "↑↓ move   enter choose format"
	case recordPlayersPhase:
		return "↑↓ move   space toggle   enter edit/continue   c add player"
	case recordChipCountsPhase:
		return "tab next   enter next player"
	case recordReviewPhase:
		if previewed {
			return "enter record game"
		}
		return "enter check balance"
	default:
		return ""
	}
}

func (m Model) recordFormatList(width int) string {
	lines := []string{brandStyle.Render("Choose a game format"), ""}
	if len(m.table.Formats) == 0 {
		return strings.Join(append(lines, mutedStyle.Render("Create a game format before recording a game.")), "\n")
	}
	for index, format := range m.table.Formats {
		style := valueStyle
		marker := "  "
		if index == m.recordFormatIndex {
			style = lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia)
			marker = "› "
		}
		lines = append(lines, style.Render(fmt.Sprintf("%s%s  ·  %s", marker, format.Name, credits(format.RequiredEntry))))
	}
	return strings.Join(lines, "\n")
}

func (m Model) recordPlayerList(width int) string {
	lines := []string{brandStyle.Render("Choose the players"), mutedStyle.Render("Select at least two players for this game."), ""}
	for index, player := range m.table.Players {
		style := valueStyle
		marker := "  "
		if index == m.playerIndex {
			style = lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia)
			marker = "› "
		}
		selected := " "
		if m.recordSelected[player.ID] {
			selected = "✓"
		}
		state := ""
		if m.recordEntered[player.ID] {
			state = "  " + lipgloss.NewStyle().Foreground(colorGreen).Render("saved")
		}
		lines = append(lines, style.Render(fmt.Sprintf("%s[%s] %s  %s%s", marker, selected, displayTablePlayerName(player, m.table.Table.HostUsername), signedCredits(player.Standing), state)))
	}
	return strings.Join(lines, "\n")
}

func (m Model) recordChipCountView() string {
	format := m.table.Formats[m.recordFormatIndex]
	player := m.table.Players[m.recordPlayerIndex]
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
	return lipgloss.JoinVertical(lipgloss.Left,
		brandStyle.Render(fmt.Sprintf("Final chips · %s", displayTablePlayerName(player, m.table.Table.HostUsername))),
		mutedStyle.Render(fmt.Sprintf("%s  ·  %s", format.Name, credits(format.RequiredEntry))),
		valueStyle.Render(fmt.Sprintf("Final value  %s  ·  P/L  %s", credits(finalValue), signedCredits(finalValue-format.RequiredEntry))),
		"", m.form.View())
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
	tableName := newCenteredInput("Table name", "", "saturday-table", &m.tableForm.name, 48, false, publicTableName).WithPrefix("#")
	m.form = huh.NewForm(huh.NewGroup(tableName)).WithTheme(huh.ThemeFunc(centeredFormTheme)).WithShowHelp(false).WithShowErrors(false)
	m.resizeForm()
}

func (m *Model) resetFormatCreateForm() {
	colors := []string{"white", "black", "green", "blue", "red", "yellow", "orange", "gray", "pink"}
	m.formatForm = &formatFormValues{chips: make([]chipFormValue, len(colors))}
	fields := []huh.Field{
		newCenteredInput("Format name", "", "saturday 2k", &m.formatForm.name, 32, false, required("enter a format name")).WithLeftAlign(true),
		newCenteredInput("Total buy-in", "", "2000", &m.formatForm.requiredEntry, 10, false, positiveIntegerText).WithLeftAlign(true),
	}
	chipSpecs := make([]chipInputSpec, 0, len(colors))
	for index, color := range colors {
		m.formatForm.chips[index].color = color
		chipSpecs = append(chipSpecs, chipInputSpec{color: color, placeholder: "value", value: &m.formatForm.chips[index].value})
	}
	fields = append(fields, newHorizontalChipInputs(chipSpecs))
	m.form = huh.NewForm(huh.NewGroup(
		fields...,
	)).WithTheme(huh.ThemeFunc(popupFormTheme)).WithShowHelp(false).WithShowErrors(false)
	m.resizeForm()
}

func (m *Model) resetPlayerCreateForm() {
	m.playerForm = &playerFormValues{}
	m.form = huh.NewForm(huh.NewGroup(
		newCenteredInput("Player name", "", "luna", &m.playerForm.name, 48, false, required("enter a player name")).WithLeftAlign(true),
		huh.NewConfirm().Title("Generate invite code").Affirmative("Yes").Negative("No").Value(&m.playerForm.generateInvite),
	)).WithTheme(huh.ThemeFunc(popupFormTheme)).WithShowHelp(false).WithShowErrors(false)
	m.resizeForm()
}

func (m *Model) resetPlayerEditForm() {
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
		newCenteredInput("Player name", "", player.Name, &m.playerForm.name, 48, false, required("enter a player name")).WithLeftAlign(true),
	}
	if m.playerInviteCodeForCurrentPlayer() == "" {
		fields = append(fields, huh.NewConfirm().Title("Generate invite code").Affirmative("Yes").Negative("No").Value(&m.playerForm.generateInvite))
	}
	m.form = huh.NewForm(huh.NewGroup(fields...)).WithTheme(huh.ThemeFunc(popupFormTheme)).WithShowHelp(false).WithShowErrors(false)
	m.resizeForm()
}

func (m *Model) resetRecordDetailsForm() {
	m.recordDetails = &recordDetailsValues{date: time.Now().Format("2006-01-02")}
	m.form = huh.NewForm(huh.NewGroup(
		newCenteredInput("Game date", "", m.recordDetails.date, &m.recordDetails.date, 10, false, isoDateText),
		newCenteredInput("Remarks", "", "optional note", &m.recordDetails.remarks, 80, false, nil),
	)).WithTheme(huh.ThemeFunc(centeredFormTheme)).WithShowHelp(false).WithShowErrors(false)
	m.resizeForm()
}

func (m *Model) resetRecordChipForm() {
	format := m.table.Formats[m.recordFormatIndex]
	player := m.table.Players[m.recordPlayerIndex]
	counts := m.recordCounts[player.ID]
	if counts == nil {
		counts = map[string]int{}
		m.recordCounts[player.ID] = counts
	}
	values := make([]string, 0, len(format.Chips))
	for _, chip := range format.Chips {
		value := strconv.Itoa(counts[chip.ID])
		values = append(values, value)
	}
	m.recordChipValues = values
	chipSpecs := make([]chipInputSpec, 0, len(format.Chips))
	for index, chip := range format.Chips {
		chipSpecs = append(chipSpecs, chipInputSpec{
			color:       chip.Color,
			placeholder: "value",
			value:       &m.recordChipValues[index],
			validate:    nonNegativeIntegerText,
		})
	}
	m.form = huh.NewForm(huh.NewGroup(newHorizontalChipInputs(chipSpecs))).WithTheme(huh.ThemeFunc(popupFormTheme)).WithShowHelp(false).WithShowErrors(false)
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
		(m.screen == recordGameScreen && (m.recordPhase == recordDetailsPhase || m.recordPhase == recordChipCountsPhase))
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

func (m Model) recordParticipants() []api.GameParticipantInput {
	format := m.table.Formats[m.recordFormatIndex]
	participants := make([]api.GameParticipantInput, 0, len(m.recordSelected))
	for _, player := range m.table.Players {
		if !m.recordSelected[player.ID] {
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
		game, err := m.api.PreviewTableGame(ctx, m.token, tableID, formatID, date, remarks, participants)
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
		table, err := m.api.RecordTableGame(ctx, m.token, tableID, formatID, date, remarks, participants)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return tableGameRecordedMsg{table: table}
	}
}
