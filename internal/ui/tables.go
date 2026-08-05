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
type tablePlayerCreatedMsg struct{ player api.TablePlayer }
type tableFormatCreatedMsg struct{ format api.GameFormat }
type tableGamePreviewedMsg struct{ game api.TableGame }
type tableGameRecordedMsg struct{ table api.TableDetail }

func (m Model) isTableScreen() bool {
	switch m.screen {
	case tablesScreen, tableDetailScreen, formatsScreen, formatDetailScreen, playersScreen,
		gamesScreen, gameDetailScreen, tableCreateScreen, formatCreateScreen, playerCreateScreen, recordGameScreen:
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
	case formatCreateScreen, formatsScreen, formatDetailScreen, playerCreateScreen, playersScreen,
		gamesScreen, gameDetailScreen:
		return tableDetailScreen
	case recordGameScreen:
		return tableDetailScreen
	default:
		return tablesScreen
	}
}

func (m Model) updateTableKey(key string) (tea.Model, tea.Cmd, bool) {
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
			if m.table.CanManage {
				m.startRecordGame()
				return m, m.form.Init(), true
			}
		case "f":
			m.screen, m.formatIndex, m.err = formatsScreen, 0, nil
			return m, nil, true
		case "p":
			m.screen, m.playerIndex, m.err = playersScreen, 0, nil
			return m, nil, true
		case "g":
			m.screen, m.gameIndex, m.err = gamesScreen, max(len(m.table.Games)-1, 0), nil
			return m, nil, true
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
		case "a", "c":
			if m.table.CanManage {
				m.screen, m.err = playerCreateScreen, nil
				m.resetPlayerCreateForm()
				return m, m.form.Init(), true
			}
		case "esc", "backspace":
			m.screen, m.err = tableDetailScreen, nil
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
			case "enter":
				if len(m.selectedPlayerIndices()) < 2 {
					m.err = errors.New("select at least two players")
					return m, nil, true
				}
				indices := m.selectedPlayerIndices()
				m.recordPlayerIndex = indices[0]
				m.recordCounts = map[string]map[string]int{}
				m.recordPhase = recordChipCountsPhase
				m.resetRecordChipForm()
				return m, m.form.Init(), true
			case "a", "c":
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
			if key == "enter" {
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

func (m Model) handleTableFormCompleted() (tea.Model, tea.Cmd) {
	switch m.screen {
	case tableCreateScreen:
		m.loading, m.status, m.err = true, "Creating table", nil
		return m, tea.Batch(m.spinner.Tick, m.createTableCmd())
	case formatCreateScreen:
		m.loading, m.status, m.err = true, "Creating game format", nil
		return m, tea.Batch(m.spinner.Tick, m.createFormatCmd())
	case playerCreateScreen:
		m.loading, m.status, m.err = true, "Adding player", nil
		return m, tea.Batch(m.spinner.Tick, m.createPlayerCmd())
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
			indices := m.selectedPlayerIndices()
			current := 0
			for index, candidate := range indices {
				if candidate == m.recordPlayerIndex {
					current = index
					break
				}
			}
			if current+1 < len(indices) {
				m.recordPlayerIndex = indices[current+1]
				m.resetRecordChipForm()
				return m, m.form.Init()
			}
			m.form, m.recordPreview, m.err = nil, nil, nil
			m.recordPhase, m.loading, m.status = recordReviewPhase, true, "Checking table balance"
			return m, tea.Batch(m.spinner.Tick, m.previewRecordGameCmd())
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

func (m *Model) startRecordGame() {
	m.screen, m.recordPhase, m.recordPreview, m.err = recordGameScreen, recordDetailsPhase, nil, nil
	m.recordSelected, m.recordCounts = map[string]bool{}, map[string]map[string]int{}
	m.recordQuickAdd, m.recordQuickAddID = false, ""
	m.resetRecordDetailsForm()
}

func (m Model) updateTableMouse(msg tableMouseMsg) (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
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
		case "search":
			m.searchActive, m.searchQuery = true, ""
		case "record":
			if m.table.CanManage {
				m.startRecordGame()
				return m, m.form.Init()
			}
		case "formats":
			m.screen = formatsScreen
		case "players":
			m.screen = playersScreen
		case "games":
			m.screen, m.gameIndex = gamesScreen, max(len(m.table.Games)-1, 0)
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
	case formatDetailScreen, gameDetailScreen:
		if msg.action == "search" && msg.activate {
			m.searchActive, m.searchQuery = true, ""
		}
	case recordGameScreen:
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
		actionY = y + lipgloss.Height(pageHeader(width, "tables", m.table.Table.Name)) + 1
		regions = actionBarHitRegions(x, actionY, items)
	case formatsScreen:
		if m.table == nil {
			return nil
		}
		items = formatActionItems(m.table.CanManage)
		actionY = y + lipgloss.Height(pageHeader(width, "tables", m.table.Table.Name, "formats")) + 1
		listY = actionY + lipgloss.Height(searchActionBar(items, m.formatActionHover, m.searchActive, m.searchQuery)) + 1
		regions = actionBarHitRegions(x, actionY, items)
		for row, index := range m.visibleFormatIndices() {
			regions = append(regions, hitRegion{x0: x, x1: x + width, y0: listY + 1 + row, y1: listY + 1 + row, value: fmt.Sprintf("format:%d", index)})
		}
	case playersScreen:
		if m.table == nil {
			return nil
		}
		items = playerActionItems(m.table.CanManage)
		actionY = y + lipgloss.Height(pageHeader(width, "tables", m.table.Table.Name, "players")) + 1
		listY = actionY + lipgloss.Height(searchActionBar(items, m.playerActionHover, m.searchActive, m.searchQuery)) + 1
		regions = actionBarHitRegions(x, actionY, items)
		for row, index := range m.visiblePlayerIndices() {
			regions = append(regions, hitRegion{x0: x, x1: x + width, y0: listY + 1 + row, y1: listY + 1 + row, value: fmt.Sprintf("player:%d", index)})
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
		actionY = y + lipgloss.Height(pageHeader(width, "tables", m.table.Table.Name, "games", m.table.Games[m.gameIndex].Date)) + 1
		regions = actionBarHitRegions(x, actionY, items)
	case gamesScreen:
		if m.table == nil {
			return nil
		}
		actionY = y + lipgloss.Height(pageHeader(width, "tables", m.table.Table.Name, "games")) + 1
		listY = actionY + lipgloss.Height(searchActionBar(gamesActionItems(), "", m.searchActive, m.searchQuery)) + 1
		visible := m.visibleGameIndices()
		for row := range visible {
			index := visible[len(visible)-1-row]
			regions = append(regions, hitRegion{x0: x, x1: x + width, y0: listY + row + 1, y1: listY + row + 1, value: fmt.Sprintf("game:%d", index)})
		}
	case recordGameScreen:
		if m.table == nil {
			return nil
		}
		baseY := y + lipgloss.Height(pageHeader(width, "tables", m.table.Table.Name, "record")) + 2
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
		{key: "g", label: "Games", action: "games"},
		{key: "f", label: "Formats", action: "formats"},
		{key: "p", label: "Players", action: "players"},
		{key: "/", label: "Search", action: "search"},
		{key: "esc", label: "Back", action: "back"},
	}
	if canManage {
		items = append([]actionBarItem{{key: "r", label: "Record game", action: "record", accent: true}}, items...)
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
		items = append([]actionBarItem{{key: "a", label: "Add player", action: "create", accent: true}}, items...)
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
	lines := []string{mutedStyle.Render("TABLE                         HOST             PLAYERS  GAMES  LAST GAME")}
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
		line := fmt.Sprintf("%s%-28s %-16s %4d     %4d  %s", marker, truncate(table.Name, 26), host, table.PlayerCount, table.GameCount, lastGame)
		lines = append(lines, style.Render(line))
	}
	return strings.Join(lines, "\n")
}

func (m Model) tableDetailView() string {
	width := max(m.width-4, 44)
	if m.loading {
		return m.pageView(lipgloss.JoinVertical(lipgloss.Left, pageHeader(width, "tables"), "", m.spinner.View()+"  "+valueStyle.Render(m.status)), tableDetailFooter())
	}
	if m.table == nil {
		return m.pageView(pageHeader(width, "tables"), tableDetailFooter())
	}
	items := tableDetailActionItems(m.table.CanManage)
	parts := []string{
		pageHeader(width, "tables", m.table.Table.Name),
		"",
		searchActionBar(items, m.tablesActionHover, m.searchActive, m.searchQuery),
		"",
		valueStyle.Render("Hosted by @"+m.table.Table.HostUsername) + " " + lipgloss.NewStyle().Foreground(colorFuchsia).Render("♛"),
		mutedStyle.Render(fmt.Sprintf("%d players  ·  %d formats  ·  %d games", len(m.table.Players), len(m.table.Formats), len(m.table.Games))),
		"",
		m.tableStandings(width),
		"",
		m.tableRecentGames(width),
	}
	if m.notice != "" {
		parts = append(parts, "", lipgloss.NewStyle().Foreground(colorGreen).Render("✓ "+m.notice))
	}
	if m.err != nil {
		parts = append(parts, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	return m.pageView(strings.Join(parts, "\n"), tableDetailFooter())
}

func tableDetailFooter() string {
	return "↑↓ move   enter select   r record   g games   f formats   p players"
}

func (m Model) tableStandings(width int) string {
	lines := []string{sectionHeading("Standings", width)}
	if len(m.table.Players) == 0 {
		return strings.Join(append(lines, mutedStyle.Render("Add players when you are ready to record a game.")), "\n")
	}
	for index, playerIndex := range m.visiblePlayerIndices() {
		player := m.table.Players[playerIndex]
		amount := standingStyle(player.Standing).Render(signedCredits(player.Standing))
		name := valueStyle.Render(truncate(player.Name, max(width-20, 12)))
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
	if m.loading {
		return m.pageView(lipgloss.JoinVertical(lipgloss.Left, pageHeader(width, "tables", m.table.Table.Name, "formats"), "", m.spinner.View()+"  "+valueStyle.Render(m.status)), formatsFooter(m.table.CanManage))
	}
	items := formatActionItems(m.table.CanManage)
	parts := []string{pageHeader(width, "tables", m.table.Table.Name, "formats"), "", searchActionBar(items, m.formatActionHover, m.searchActive, m.searchQuery), "", m.formatList(width)}
	if m.err != nil {
		parts = append(parts, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	return m.pageView(strings.Join(parts, "\n"), formatsFooter(m.table.CanManage))
}

func formatsFooter(canManage bool) string {
	if canManage {
		return "↑↓ move   enter inspect   c create"
	}
	return "↑↓ move   enter inspect"
}

func (m Model) formatList(width int) string {
	if len(m.table.Formats) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, valueStyle.Render("No game formats yet"), mutedStyle.Render("The table host can add the first format."))
	}
	if len(m.visibleFormatIndices()) == 0 {
		return mutedStyle.Render("No formats match the search.")
	}
	lines := []string{mutedStyle.Render("FORMAT                                      ENTRY       CHIPS")}
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
			chipNames = append(chipNames, fmt.Sprintf("%s %d", chip.Label, chip.Value))
		}
		line := fmt.Sprintf("%s%-40s %-10s %s", marker, truncate(format.Name, 38), credits(format.RequiredEntry), strings.Join(chipNames, ", "))
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
		lines = append(lines, chipStyle(chip.Color).Render("● "+chip.Label)+"  "+valueStyle.Render(credits(chip.Value)))
	}
	return m.pageView(strings.Join(lines, "\n"), "")
}

func (m Model) playersView() string {
	width := max(m.width-4, 44)
	if m.table == nil {
		return m.pageView(pageHeader(width, "tables", "players"), playersFooter(false))
	}
	if m.loading {
		return m.pageView(lipgloss.JoinVertical(lipgloss.Left, pageHeader(width, m.table.Table.Name, "players"), "", m.spinner.View()+"  "+valueStyle.Render(m.status)), playersFooter(m.table.CanManage))
	}
	items := playerActionItems(m.table.CanManage)
	parts := []string{pageHeader(width, "tables", m.table.Table.Name, "players"), "", searchActionBar(items, m.playerActionHover, m.searchActive, m.searchQuery), "", m.playerList(width)}
	if m.err != nil {
		parts = append(parts, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	return m.pageView(strings.Join(parts, "\n"), playersFooter(m.table.CanManage))
}

func playersFooter(canManage bool) string {
	if canManage {
		return "↑↓ move   a add player"
	}
	return "↑↓ move"
}

func (m Model) gamesView() string {
	width := max(m.width-4, 44)
	if m.table == nil {
		return m.pageView(pageHeader(width, "tables", "games"), "")
	}
	if m.loading {
		return m.pageView(lipgloss.JoinVertical(lipgloss.Left,
			pageHeader(width, "tables", m.table.Table.Name, "games"), "",
			m.spinner.View()+"  "+valueStyle.Render(m.status)), gamesFooter())
	}
	items := gamesActionItems()
	parts := []string{pageHeader(width, "tables", m.table.Table.Name, "games"), "", searchActionBar(items, "", m.searchActive, m.searchQuery), "", m.gameList(width)}
	if m.err != nil {
		parts = append(parts, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	return m.pageView(strings.Join(parts, "\n"), gamesFooter())
}

func gamesFooter() string {
	return "↑↓ move   enter inspect"
}

func gamesActionItems() []actionBarItem {
	return []actionBarItem{{key: "/", label: "Search", action: "search"}, {key: "esc", label: "Back", action: "back"}}
}

func (m Model) gameList(width int) string {
	if len(m.table.Games) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			valueStyle.Render("No games recorded yet"),
			mutedStyle.Render("Recorded games will stay here with their chip snapshots."))
	}
	if len(m.visibleGameIndices()) == 0 {
		return mutedStyle.Render("No games match the search.")
	}
	lines := []string{mutedStyle.Render("DATE          FORMAT                              PLAYERS  STATUS")}
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
		line := fmt.Sprintf("%s%-12s %-36s %7d  %s", marker, game.Date,
			truncate(game.Format.Name, 34), len(game.Participants), statusBadge(game.Status))
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
		pageHeader(width, "tables", m.table.Table.Name, "games", game.Date),
		"",
		searchActionBar([]actionBarItem{{key: "/", label: "Search", action: "search"}, {key: "esc", label: "Back", action: "back"}}, "", m.searchActive, m.searchQuery),
		"",
		valueStyle.Render(game.Format.Name),
		mutedStyle.Render(fmt.Sprintf("%s  ·  required entry %s  ·  %s", game.Date, credits(game.Format.RequiredEntry), statusBadge(game.Status))),
		"",
		sectionHeading("Chip snapshot", width),
	}
	for _, chip := range game.Format.Chips {
		lines = append(lines, "  "+chipStyle(chip.Color).Render("● "+chip.Label)+"  "+valueStyle.Render(credits(chip.Value)))
	}
	lines = append(lines, "", sectionHeading("Results", width))
	for _, participant := range game.Participants {
		if !searchMatches(m.searchQuery, participant.PlayerName) {
			continue
		}
		lines = append(lines, valueStyle.Render(truncate(participant.PlayerName, 22))+"  "+
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
		return lipgloss.JoinVertical(lipgloss.Left, valueStyle.Render("No players yet"), mutedStyle.Render("Add a player before recording a game."))
	}
	if len(m.visiblePlayerIndices()) == 0 {
		return mutedStyle.Render("No players match the search.")
	}
	lines := []string{mutedStyle.Render("PLAYER" + strings.Repeat(" ", max(width-18, 4)) + "STANDING")}
	for _, index := range m.visiblePlayerIndices() {
		player := m.table.Players[index]
		marker := "  "
		style := valueStyle
		if index == m.playerIndex {
			marker = "› "
			style = lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia)
		}
		name := truncate(player.Name, max(width-18, 8))
		line := fmt.Sprintf("%s%-*s %s", marker, max(width-4, 14), name, signedCredits(player.Standing))
		lines = append(lines, style.Render(line))
	}
	return strings.Join(lines, "\n")
}

func (m Model) tableCreateView() string {
	return m.centeredFormPage("tables", "Create a table", "Create", "")
}

func (m Model) formatCreateView() string {
	if m.table == nil {
		return m.pageView(pageHeader(max(m.width-4, 44), "formats"), "")
	}
	return m.centeredFormPage(strings.Join([]string{"tables", m.table.Table.Name, "formats"}, " / "), "Create game format", "Create", "")
}

func (m Model) playerCreateView() string {
	if m.table == nil {
		return m.pageView(pageHeader(max(m.width-4, 44), "players"), "")
	}
	crumb := []string{"tables", m.table.Table.Name, "players"}
	if m.recordQuickAdd {
		crumb = []string{"tables", m.table.Table.Name, "record", "add player"}
	}
	return m.centeredFormPage(strings.Join(crumb, " / "), "Add player", "Create", "")
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
	header := pageHeader(width, "tables", m.table.Table.Name, "record")
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
		body = m.recordChipCountView()
	case recordReviewPhase:
		body = m.recordReviewView(width)
	}
	if m.err != nil {
		body = lipgloss.JoinVertical(lipgloss.Center, body, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	return m.pageView(lipgloss.JoinVertical(lipgloss.Left, header, "", body), recordFooter(m.recordPhase, m.recordPreview != nil))
}

func recordFooter(phase recordPhase, previewed bool) string {
	switch phase {
	case recordDetailsPhase:
		return "tab next   enter continue"
	case recordFormatPhase:
		return "↑↓ move   enter choose format"
	case recordPlayersPhase:
		return "↑↓ move   space toggle   enter continue"
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
		lines = append(lines, style.Render(fmt.Sprintf("%s[%s] %s  %s", marker, selected, player.Name, signedCredits(player.Standing))))
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
	return lipgloss.JoinVertical(lipgloss.Center,
		brandStyle.Render(fmt.Sprintf("Final chips · %s", player.Name)),
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
		lines = append(lines, fmt.Sprintf("%-18s %8s  %s", truncate(participant.PlayerName, 18), signedCredits(participant.ProfitLoss), standingStyle(participant.EndingStanding).Render(signedCredits(participant.EndingStanding))))
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
	case "purple":
		value = colorFuchsia
	}
	return lipgloss.NewStyle().Foreground(value)
}

func (m *Model) resetTableCreateForm() {
	m.tableForm = &tableFormValues{}
	tableName := newCenteredInput("Table name", "", "saturday-table", &m.tableForm.name, 48, false, publicTableName).WithPrefix("#")
	m.form = huh.NewForm(huh.NewGroup(tableName)).WithTheme(huh.ThemeFunc(centeredFormTheme)).WithShowHelp(false).WithShowErrors(false)
	m.resizeForm()
}

func (m *Model) resetFormatCreateForm() {
	m.formatForm = &formatFormValues{}
	m.form = huh.NewForm(huh.NewGroup(
		newCenteredInput("Format name", "", "Saturday 2K", &m.formatForm.name, 48, false, required("Enter a format name")),
		newCenteredInput("Required entry", "", "2000", &m.formatForm.requiredEntry, 10, false, positiveIntegerText),
		newCenteredInput("Chips", "", "white=10, black=100", &m.formatForm.chips, 160, false, chipSpec),
	)).WithTheme(huh.ThemeFunc(centeredFormTheme)).WithShowHelp(false).WithShowErrors(false)
	m.resizeForm()
}

func (m *Model) resetPlayerCreateForm() {
	m.playerForm = &playerFormValues{}
	m.form = huh.NewForm(huh.NewGroup(newCenteredInput("Player name", "", "Tahseen", &m.playerForm.name, 48, false, required("Enter a player name")))).WithTheme(huh.ThemeFunc(centeredFormTheme)).WithShowHelp(false).WithShowErrors(false)
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
	fields := make([]*centeredInput, 0, len(format.Chips))
	for index, chip := range format.Chips {
		fields = append(fields, newCenteredInput(chip.Label, "", fmt.Sprintf("value %d", chip.Value), &m.recordChipValues[index], 10, false, nonNegativeIntegerText))
	}
	groups := make([]huh.Field, 0, len(fields))
	for _, field := range fields {
		groups = append(groups, field)
	}
	m.form = huh.NewForm(huh.NewGroup(groups...)).WithTheme(huh.ThemeFunc(centeredFormTheme)).WithShowHelp(false).WithShowErrors(false)
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

func (m Model) isTableFormScreen() bool {
	return m.screen == tableCreateScreen || m.screen == formatCreateScreen || m.screen == playerCreateScreen ||
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
	tableID, name := m.table.Table.ID, m.playerForm.name
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		player, err := m.api.CreateTablePlayer(ctx, m.token, tableID, name)
		if err != nil {
			return operationFailedMsg{err: err}
		}
		return tablePlayerCreatedMsg{player: player}
	}
}

func (m Model) createFormatCmd() tea.Cmd {
	tableID, form := m.table.Table.ID, *m.formatForm
	entry, _ := strconv.Atoi(strings.TrimSpace(form.requiredEntry))
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		format, err := m.api.CreateGameFormat(ctx, m.token, tableID, form.name, entry, parseChipSpec(form.chips))
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
