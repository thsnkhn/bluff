package ui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/thsnkhn/bluff/internal/api"
)

type tableSection int

const (
	tableOverviewSection tableSection = iota
	tableGamesSection
	tablePlayersSection
	tableFormatsSection
)

type tableWorkspaceMetrics struct {
	width        int
	actionY      int
	workspaceY   int
	sidebarWidth int
	contentWidth int
}

func (m Model) tableWorkspaceNavRegions(metrics tableWorkspaceMetrics) []hitRegion {
	prefixWidth := lipgloss.Width("~bluff / " + m.table.Table.Name)
	// Header layout: ~bluff / #table /// OVERVIEW | PLAYERS ...
	cursor := 2 + prefixWidth + 1 + 3 + 1
	labels := []struct {
		label  string
		action string
	}{
		{" OVERVIEW ", "nav:overview"},
		{" GAMES ", "nav:games"},
		{" PLAYERS ", "nav:players"},
		{" FORMATS ", "nav:formats"},
	}
	regions := make([]hitRegion, 0, len(labels))
	for index, item := range labels {
		itemWidth := lipgloss.Width(item.label)
		regions = append(regions, hitRegion{
			x0: cursor, x1: cursor + itemWidth - 1,
			y0: 1, y1: 1,
			value: item.action,
		})
		cursor += itemWidth
		if index < len(labels)-1 {
			cursor++ // separator pipe
		}
	}
	return regions
}

func (m Model) currentTableSection() tableSection {
	switch m.screen {
	case playersScreen:
		return tablePlayersSection
	case playerDetailScreen:
		return tablePlayersSection
	case formatsScreen, formatDetailScreen:
		return tableFormatsSection
	case gamesScreen, gameDetailScreen:
		return tableGamesSection
	default:
		return tableOverviewSection
	}
}

func (m Model) tableWorkspaceMetrics(width int, items []actionBarItem) tableWorkspaceMetrics {
	actionY := 1 + lipgloss.Height(m.tableWorkspaceHeader(width)) + 1
	workspaceY := actionY + lipgloss.Height(searchActionBar(items, "", false, "")) + 1
	contentWidth := max(width-4, 18)
	return tableWorkspaceMetrics{
		width:        width,
		actionY:      actionY,
		workspaceY:   workspaceY,
		sidebarWidth: 0,
		contentWidth: contentWidth,
	}
}

func workspaceContentWidth(metrics tableWorkspaceMetrics) int {
	// Subpages share the same content edge and width as the top-level tables list.
	return max(metrics.contentWidth, 14)
}

func workspaceEmptyState(width int, title, description string) string {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Left).Render(
		lipgloss.JoinVertical(lipgloss.Left, valueStyle.Render(title), mutedStyle.Render(description)),
	)
}

func (m Model) tableWorkspace(active tableSection, items []actionBarItem, content string, footer string) string {
	width := max(m.width-4, 44)
	metrics := m.tableWorkspaceMetrics(width, items)
	header := m.tableWorkspaceHeader(width)
	actions := searchActionBar(items, m.tablesActionHover, m.searchActive, m.searchQuery)
	if active == tablePlayersSection {
		actions = searchActionBar(items, m.playerActionHover, m.searchActive, m.searchQuery)
	}
	if active == tableFormatsSection {
		actions = searchActionBar(items, m.formatActionHover, m.searchActive, m.searchQuery)
	}
	if active == tableGamesSection {
		actions = searchActionBar(items, "", m.searchActive, m.searchQuery)
	}

	main := lipgloss.NewStyle().Width(metrics.contentWidth).Render(content)
	parts := []string{header, ""}
	if active != tableOverviewSection {
		parts = append(parts, actions, "")
	}
	parts = append(parts, main)
	if m.err != nil {
		parts = append(parts, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	return m.pageView(strings.Join(parts, "\n"), footer)
}

func (m Model) tableWorkspaceHeader(width int) string {
	prefix := lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia).Render("~bluff") +
		mutedStyle.Render(" / ") + valueStyle.Render(m.table.Table.Name)
	active := m.currentTableSection()
	sections := []struct {
		section tableSection
		label   string
	}{
		{tableOverviewSection, "OVERVIEW"},
		{tableGamesSection, "GAMES"},
		{tablePlayersSection, "PLAYERS"},
		{tableFormatsSection, "FORMATS"},
	}
	tabs := make([]string, 0, len(sections))
	for _, item := range sections {
		label := " " + item.label + " "
		if item.section == active {
			tabs = append(tabs, lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia).Render(label))
		} else {
			tabs = append(tabs, mutedStyle.Render(label))
		}
	}
	tabLine := strings.Join(tabs, mutedStyle.Render("|"))
	slashes := lipgloss.NewStyle().Foreground(colorIndigo).Render("///")
	used := lipgloss.Width(prefix) + 1 + lipgloss.Width(slashes) + lipgloss.Width(tabLine)
	ruleWidth := max(width-used, 4)
	return prefix + " " + slashes + tabLine + " " + lipgloss.NewStyle().Foreground(colorIndigo).Render(strings.Repeat("/", ruleWidth))
}

func (m Model) tableSidebar(width int, active, focused tableSection) string {
	focusedStyle := lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia).Width(max(width-2, 1))
	normal := valueStyle.Width(max(width-2, 1))
	nav := []struct {
		section tableSection
		label   string
	}{
		{tableGamesSection, "Games"},
		{tablePlayersSection, "Players"},
		{tableFormatsSection, "Formats"},
	}
	lines := []string{
		brandStyle.Width(max(width-2, 1)).Render(m.table.Table.Name),
		"",
	}
	for _, item := range nav {
		label := "  " + item.label
		if item.section == focused {
			lines = append(lines, focusedStyle.Render("› "+item.label))
		} else {
			lines = append(lines, normal.Render(label))
		}
	}
	return lipgloss.NewStyle().Width(width).Padding(1, 1).Render(strings.Join(lines, "\n"))
}

func (m Model) tableOverviewContent(width int) string {
	stats := m.tableOverviewStats(width)
	statsBox := lipgloss.NewStyle().Width(width).Padding(0, 1).
		Border(lipgloss.NormalBorder()).BorderForeground(colorMuted).Render(stats)
	return lipgloss.JoinVertical(lipgloss.Left,
		statsBox,
		"",
		m.tableStandingChart(width),
		"",
		m.tableChipChart(width),
	)
}

func (m Model) tableOverviewStats(width int) string {
	tableValue := 0
	for _, player := range m.table.Players {
		tableValue += player.Standing
	}
	host := "@" + m.table.Table.HostUsername + " " + lipgloss.NewStyle().Foreground(colorFuchsia).Render("♛")
	stats := []string{
		workspaceStat("HOST", host),
		workspaceStat("PLAYERS", fmt.Sprintf("%d", len(m.table.Players))),
		workspaceStat("GAMES", fmt.Sprintf("%d", len(m.table.Games))),
		workspaceStat("TABLE VALUE", standingStyle(tableValue).Render(signedCredits(tableValue))),
	}
	if width < 64 {
		return lipgloss.NewStyle().Align(lipgloss.Center).Render(lipgloss.JoinVertical(lipgloss.Center, stats...))
	}
	columnWidth := max((width-6)/4, 12)
	columns := make([]string, 0, len(stats))
	for _, stat := range stats {
		columns = append(columns, lipgloss.NewStyle().Width(columnWidth).Align(lipgloss.Center).Render(stat))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, columns[0], "  ", columns[1], "  ", columns[2], "  ", columns[3])
}

func workspaceStat(label, value string) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		mutedStyle.Render(label),
		valueStyle.Render(value),
	)
}

func (m Model) tableStandingChart(width int) string {
	players := sortedTablePlayers(m.table.Players)
	lines := []string{brandStyle.Render("Player standings")}
	if len(players) == 0 || len(m.table.Games) == 0 {
		return strings.Join(append(lines, "", mutedStyle.Render("Record a game to see date-wise standings.")), "\n")
	}

	games := append([]api.TableGame(nil), m.table.Games...)
	sort.SliceStable(games, func(i, j int) bool { return games[i].Date < games[j].Date })
	values := make(map[string][]int, len(players))
	maximum := 1
	for _, player := range players {
		standing := 0
		values[player.ID] = make([]int, 0, len(games))
		for _, game := range games {
			for _, participant := range game.Participants {
				if participant.PlayerID == player.ID {
					standing = participant.EndingStanding
					break
				}
			}
			values[player.ID] = append(values[player.ID], standing)
			maximum = max(maximum, abs(standing))
		}
	}

	nameWidth := min(max(width/5, 10), 18)
	plotWidth := max(width-nameWidth-2, 16)
	cellWidth := max((plotWidth-max(len(games)-1, 0))/len(games), 7)
	dateCells := make([]string, 0, len(games))
	for _, game := range games {
		dateCells = append(dateCells, lipgloss.NewStyle().Width(cellWidth).Align(lipgloss.Center).Render(truncate(game.Date, cellWidth)))
	}
	lines = append(lines, strings.Repeat(" ", nameWidth+2)+strings.Join(dateCells, " "))
	zeroCells := make([]string, 0, len(games))
	for range games {
		zeroCells = append(zeroCells, lipgloss.NewStyle().Width(cellWidth).Align(lipgloss.Center).Render("0"))
	}
	lines = append(lines, mutedStyle.Render(strings.Repeat(" ", nameWidth+2)+strings.Join(zeroCells, " ")))
	axis := strings.Repeat("─", cellWidth)
	lines = append(lines, mutedStyle.Render(strings.Repeat(" ", nameWidth+2)+strings.Join(repeatString(axis, len(games)), " ")))
	for _, player := range players {
		name := truncate(displayTablePlayerName(player, m.table.Table.HostUsername), max(nameWidth-2, 1))
		if strings.EqualFold(player.Name, m.table.Table.HostUsername) {
			name += " " + lipgloss.NewStyle().Foreground(colorFuchsia).Render("♛")
		}
		nameCell := lipgloss.NewStyle().Width(nameWidth).Render(name)
		cells := make([]string, 0, len(games))
		for _, value := range values[player.ID] {
			cells = append(cells, standingChartCell(value, maximum, cellWidth))
		}
		lines = append(lines, nameCell+"  "+strings.Join(cells, " "))
	}
	return strings.Join(lines, "\n")
}

func repeatString(value string, count int) []string {
	result := make([]string, count)
	for index := range result {
		result[index] = value
	}
	return result
}

func standingChartCell(value, maximum, width int) string {
	canvas := []rune(strings.Repeat(" ", width))
	axis := width / 2
	canvas[axis] = '│'
	bar := abs(value) * max(width-axis-1, 1) / max(maximum, 1)
	if value > 0 {
		for index := axis + 1; index <= min(axis+bar, width-1); index++ {
			canvas[index] = '━'
		}
	} else if value < 0 {
		for index := max(axis-bar, 0); index < axis; index++ {
			canvas[index] = '━'
		}
	}
	return standingStyle(value).Render(string(canvas))
}

func (m Model) tableChipChart(width int) string {
	lines := []string{brandStyle.Render("Chip values")}
	players := sortedTablePlayers(m.table.Players)
	if len(players) == 0 {
		return strings.Join(append(lines, "", mutedStyle.Render("No player values yet.")), "\n")
	}
	maximum := 1
	for _, player := range players {
		maximum = max(maximum, abs(player.Standing))
	}
	nameWidth := min(max(width/3, 8), 16)
	valueWidth := 9
	plotWidth := max(width-nameWidth-valueWidth-3, 12)
	axis := plotWidth / 2
	leftWidth, rightWidth := axis, plotWidth-axis-1
	for _, player := range players {
		name := truncate(displayTablePlayerName(player, m.table.Table.HostUsername), nameWidth-2)
		if strings.EqualFold(player.Name, m.table.Table.HostUsername) {
			name += " " + lipgloss.NewStyle().Foreground(colorFuchsia).Render("♛")
		}
		name = lipgloss.NewStyle().Width(nameWidth).Render(name)
		bar := make([]rune, plotWidth)
		for index := range bar {
			bar[index] = ' '
		}
		bar[axis] = '┆'
		fill := abs(player.Standing) * max(leftWidth, rightWidth) / maximum
		if player.Standing > 0 {
			for index := axis + 1; index <= min(axis+fill, plotWidth-1); index++ {
				bar[index] = '█'
			}
		} else if player.Standing < 0 {
			for index := max(axis-fill, 0); index < axis; index++ {
				bar[index] = '█'
			}
		}
		barColor := colorIndigo
		if player.Standing > 0 {
			barColor = colorFuchsia
		}
		barView := lipgloss.NewStyle().Foreground(barColor).Render(string(bar))
		value := standingStyle(player.Standing).Width(valueWidth).Align(lipgloss.Right).Render(fmt.Sprintf("%d", player.Standing))
		lines = append(lines, name+barView+value)
	}
	return strings.Join(lines, "\n")
}

func displayTablePlayerName(player api.TablePlayer, hostUsername string) string {
	username := strings.TrimPrefix(strings.TrimSpace(player.Username), "@")
	if username == "" && strings.EqualFold(player.Name, hostUsername) {
		username = strings.TrimPrefix(strings.TrimSpace(player.Name), "@")
	}
	if username != "" {
		return "@" + username
	}
	return player.Name
}

func displayParticipantName(name, hostUsername string) string {
	name = strings.TrimSpace(name)
	if name != "" && strings.EqualFold(strings.TrimPrefix(name, "@"), strings.TrimPrefix(hostUsername, "@")) {
		return "@" + strings.TrimPrefix(name, "@")
	}
	return name
}

type tableChipSummary struct {
	label string
	color string
	count int
}

func (m Model) tableChipSummaries() []tableChipSummary {
	counts := map[string]tableChipSummary{}
	for _, game := range m.table.Games {
		for _, participant := range game.Participants {
			for _, chip := range participant.ChipCounts {
				key := chip.Label + ":" + chip.Color
				summary := counts[key]
				summary.label, summary.color = chip.Label, chip.Color
				summary.count += chip.Count
				counts[key] = summary
			}
		}
	}
	result := make([]tableChipSummary, 0, len(counts))
	for _, chip := range counts {
		result = append(result, chip)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].count == result[j].count {
			return result[i].label < result[j].label
		}
		return result[i].count > result[j].count
	})
	if len(result) > 6 {
		result = result[:6]
	}
	return result
}

func sortedTablePlayers(players []api.TablePlayer) []api.TablePlayer {
	result := append([]api.TablePlayer(nil), players...)
	sort.SliceStable(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
