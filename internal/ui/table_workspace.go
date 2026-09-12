package ui

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"

	"github.com/thsnkhn/bluff/internal/api"
)

type tableSection int

const (
	tableOverviewSection tableSection = iota
	tableGamesSection
	tablePlayersSection
	tableFormatsSection
)

type tableChartFocus int

const (
	tableChartNone tableChartFocus = iota
	tableChartHistory
	tableChartStandings
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
	case formatsScreen:
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
	content := lipgloss.NewStyle().Width(width).Align(lipgloss.Left).Render(
		lipgloss.JoinVertical(lipgloss.Left, valueStyle.Render(title), mutedStyle.Render(description)),
	)
	return spacedEmptyState(content)
}

func spacedEmptyState(content string) string { return "\n" + content }

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

	mainWidth := metrics.contentWidth
	if active == tableOverviewSection {
		// The overview stats bar is a full-width frame. Give the overview the
		// complete workspace width instead of the narrower list inset.
		mainWidth = metrics.width
	}
	main := lipgloss.NewStyle().Width(mainWidth).Render(content)
	parts := []string{header, ""}
	if active != tableOverviewSection {
		parts = append(parts, actions)
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
	if m.expandedChart == tableChartHistory {
		return m.tableStandingChartSized(width, m.expandedStandingChartHeight(), "esc back")
	}
	if m.expandedChart == tableChartStandings {
		return m.tableChipChartWithShortcut(width, m.standingEntryShortcuts())
	}
	stats := m.tableOverviewStats(width)
	statsBox := lipgloss.NewStyle().Width(width).Padding(0, 1).
		Border(lipgloss.NormalBorder()).BorderForeground(colorMuted).Render(stats)
	if len(m.table.Games) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			statsBox,
			"",
			workspaceEmptyState(width, "Record a game", "Stats will appear here after the first game."),
		)
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		statsBox,
		"",
		m.tableChipChart(width),
		"",
		m.tableStandingChart(width),
	)
}

func (m Model) tableOverviewStats(width int) string {
	host := "@" + m.table.Table.HostUsername + " " + lipgloss.NewStyle().Foreground(colorFuchsia).Render("♛")
	stats := []string{
		workspaceStat("HOST", host),
		workspaceStat("PLAYERS", fmt.Sprintf("%d", len(m.table.Players))),
		workspaceStat("GAMES", fmt.Sprintf("%d", len(m.table.Games))),
	}
	if width < 64 {
		return lipgloss.NewStyle().Align(lipgloss.Center).Render(lipgloss.JoinVertical(lipgloss.Center, stats...))
	}
	columnWidth := max((width-4)/3, 12)
	columns := make([]string, 0, len(stats))
	for _, stat := range stats {
		columns = append(columns, lipgloss.NewStyle().Width(columnWidth).Align(lipgloss.Center).Render(stat))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, columns[0], "  ", columns[1], "  ", columns[2])
}

func workspaceStat(label, value string) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		mutedStyle.Render(label),
		valueStyle.Render(value),
	)
}

func (m Model) tableStandingChart(width int) string {
	return m.tableStandingChartSized(width, 16, "alt+h expand")
}

func (m Model) tableStandingChartSized(width, chartHeight int, shortcut string) string {
	players := sortedTablePlayers(m.table.Players)
	lines := []string{chartSectionHeadingWithShortcut("History", shortcut, width), ""}
	if len(players) == 0 || len(m.table.Games) == 0 {
		return strings.Join(append(lines, mutedStyle.Render("Record a game to see player history.")), "\n")
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

	maxX := float64(max(len(games)-1, 1))
	axisStyle := lipgloss.NewStyle().Foreground(colorMuted)
	chart := linechart.New(max(width, 24), chartHeight, 0, maxX, -float64(maximum), float64(maximum),
		linechart.WithXYSteps(0, 2),
		linechart.WithYLabelFormatter(func(_ int, value float64) string {
			return credits(int(value))
		}),
		linechart.WithStyles(axisStyle, axisStyle, axisStyle),
	)
	chart.DrawXYAxisAndLabel()
	chart.DrawRuneLineWithStyle(
		canvas.Float64Point{X: 0, Y: 0},
		canvas.Float64Point{X: maxX, Y: 0},
		'─',
		axisStyle,
	)
	palette := standingChartPalette()
	for playerIndex, player := range players {
		color := palette[playerIndex%len(palette)]
		style := lipgloss.NewStyle().Foreground(color)
		points := make([]canvas.Float64Point, 0, len(games))
		for gameIndex, value := range values[player.ID] {
			points = append(points, canvas.Float64Point{X: float64(gameIndex), Y: float64(value)})
		}
		for index := 1; index < len(points); index++ {
			chart.DrawBrailleLineWithStyle(points[index-1], points[index], style)
		}
		for _, point := range points {
			chart.DrawRuneWithStyle(point, '◆', style)
		}
	}
	lines = append(lines, chart.View())

	labelWidth := max(len(credits(maximum)), len(credits(-maximum))) + 1
	plotWidth := max(width-labelWidth, 1)
	dots := make([]rune, plotWidth)
	for index := range dots {
		dots[index] = ' '
	}
	for gameIndex := range games {
		x := 0
		if len(games) > 1 {
			x = gameIndex * (plotWidth - 1) / (len(games) - 1)
		}
		dots[x] = '•'
	}
	lines = append(lines, strings.Repeat(" ", labelWidth+2)+mutedStyle.Render(string(dots)))
	legends := make([]string, 0, len(players))
	for index, player := range players {
		color := palette[index%len(palette)]
		name := displayTablePlayerName(player)
		legends = append(legends, lipgloss.NewStyle().Foreground(color).Render("◆ "+name))
	}
	lines = append(lines, "", strings.Join(legends, "   "))
	return strings.Join(lines, "\n")
}

func standingChartPalette() []color.Color {
	return []color.Color{
		colorFuchsia,
		colorIndigo,
		colorGreen,
		lipgloss.Color("#FFB86C"),
		lipgloss.Color("#FFD866"),
		lipgloss.Color("#78DCE8"),
	}
}

func (m Model) standingEntryShortcuts() string {
	shortcuts := []string{}
	last := max(len(m.table.Games)-1, 0)
	offset := min(max(m.standingEntryOffset, 0), last)
	if offset < last {
		shortcuts = append(shortcuts, "← prev")
	}
	if offset > 0 {
		shortcuts = append(shortcuts, "→ next")
	}
	return strings.Join(append(shortcuts, "esc back"), "   ")
}

func (m Model) tableChipChart(width int) string {
	return m.tableChipChartWithShortcut(width, "alt+s expand")
}

func (m Model) tableChipChartWithShortcut(width int, shortcut string) string {
	entryLabel := ""
	if m.expandedChart == tableChartStandings && len(m.table.Games) > 0 {
		snapshot := *m.table
		m.table = &snapshot
		games := append([]api.TableGame(nil), m.table.Games...)
		sort.SliceStable(games, func(i, j int) bool { return games[i].Date < games[j].Date })
		entry := len(games) - 1 - min(max(m.standingEntryOffset, 0), len(games)-1)
		entryLabel = fmt.Sprintf("%d/%d · %s", entry+1, len(games), games[entry].Date)
		if entry == len(games)-1 {
			entryLabel += " · " + brandStyle.Foreground(colorGreen).Render("Latest")
		}
		// Reverse later entries to preserve each player's carried balance.
		// TODO: Share this snapshot calculation if other history views need it.
		m.table.Players = append([]api.TablePlayer(nil), m.table.Players...)
		for index := len(games) - 1; index > entry; index-- {
			for _, participant := range games[index].Participants {
				for playerIndex := range m.table.Players {
					if m.table.Players[playerIndex].ID == participant.PlayerID {
						m.table.Players[playerIndex].Standing = participant.StartingStanding
					}
				}
			}
		}
		m.table.Games = games[:entry+1]
	}
	title := "Standings"
	if entryLabel != "" {
		title += " · " + entryLabel
	}
	lines := []string{chartSectionHeadingWithShortcut(title, shortcut, width), ""}
	players := append([]api.TablePlayer(nil), m.table.Players...)
	sort.SliceStable(players, func(i, j int) bool {
		if players[i].Standing == players[j].Standing {
			return strings.ToLower(players[i].Name) < strings.ToLower(players[j].Name)
		}
		return players[i].Standing > players[j].Standing
	})
	if len(players) == 0 {
		return strings.Join(append(lines, "", mutedStyle.Render("No player values yet.")), "\n")
	}
	maximum := 1
	for _, player := range players {
		maximum = max(maximum, abs(player.Standing))
	}
	ledger := m.tablePlayerLedgerSummaries()
	previous := append([]api.TablePlayer(nil), players...)
	for i := range previous {
		previous[i].Standing -= ledger[previous[i].ID].change
	}
	ranks, oldRanks := standingRanks(players), standingRanks(previous)
	showLedger := width >= 70
	showInsights := width >= 110
	nameWidth := min(max(width/5, 8), 16)
	rankWidth := 9
	valueWidth := 11
	type column struct {
		title string
		width int
	}
	columns := []column{}
	if showLedger {
		columns = append(columns, column{"CHANGE", 11})
	}
	if showInsights {
		columns = append(columns, column{"AVERAGE", 11}, column{"WIN %", 6})
	}
	if showLedger {
		columns = append(columns, column{"GAMES", 7})
	}
	if showInsights {
		columns = append(columns, column{"STREAK", 8})
	}
	ledgerWidth := 0
	for _, col := range columns {
		ledgerWidth += col.width + 1
	}
	const chartGap = "   "
	plotWidth := max(width-rankWidth-nameWidth-valueWidth-ledgerWidth-len(chartGap), 4)
	header := mutedStyle.Width(rankWidth).Render("RANK") + strings.Repeat(" ", nameWidth) + mutedStyle.Width(valueWidth).Align(lipgloss.Right).Render("BALANCE")
	for _, col := range columns {
		header += " " + mutedStyle.Width(col.width).Align(lipgloss.Right).Render(col.title)
	}
	lines = append(lines, header)
	axis := plotWidth / 2
	leftWidth, rightWidth := axis, plotWidth-axis-1
	for playerIndex, player := range players {
		name := truncate(displayTablePlayerName(player), nameWidth-2)
		if tablePlayerIsHost(player, m.table.Table) {
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
		barColor := colorMuted
		if player.Standing > 0 {
			barColor = colorGreen
		} else if player.Standing < 0 {
			barColor = colorRed
		}
		barView := lipgloss.NewStyle().Foreground(barColor).Render(string(bar))
		value := standingStyle(player.Standing).Width(valueWidth).Align(lipgloss.Right).Render(signedCredits(player.Standing))
		summary := ledger[player.ID]
		movement := oldRanks[player.ID] - ranks[player.ID]
		marker := mutedStyle.Render("—")
		if movement > 0 {
			marker = standingStyle(1).Render(fmt.Sprintf("↑%d", movement))
		}
		if movement < 0 {
			marker = standingStyle(-1).Render(fmt.Sprintf("↓%d", -movement))
		}
		rank := valueStyle.Render(fmt.Sprintf("%d ", ranks[player.ID])) + marker
		row := lipgloss.NewStyle().Width(rankWidth).Render(rank) + name + value
		if showLedger {
			row += " " + standingStyle(summary.change).Width(11).Align(lipgloss.Right).Render(signedCredits(summary.change))
		}
		if showInsights {
			average, winRate, streak := "—", "—", "—"
			if summary.games > 0 {
				average = fmt.Sprintf("%+.0f cr", float64(summary.withdrawn-summary.buyIn)/float64(summary.games))
				winRate = fmt.Sprintf("%.0f%%", 100*float64(summary.wins)/float64(summary.games))
			}
			if summary.streak > 0 {
				streak = fmt.Sprintf("%dW", summary.streak)
			}
			if summary.streak < 0 {
				streak = fmt.Sprintf("%dL", -summary.streak)
			}
			row += " " + standingStyle(summary.withdrawn-summary.buyIn).Width(11).Align(lipgloss.Right).Render(average) +
				" " + valueStyle.Width(6).Align(lipgloss.Right).Render(winRate) +
				" " + valueStyle.Width(7).Align(lipgloss.Right).Render(fmt.Sprint(summary.games)) +
				" " + standingStyle(summary.streak).Width(8).Align(lipgloss.Right).Render(streak)
		} else if showLedger {
			row += " " + valueStyle.Width(7).Align(lipgloss.Right).Render(fmt.Sprint(summary.games))
		}
		lines = append(lines, row+chartGap+barView)
		if m.expandedChart == tableChartStandings && playerIndex < len(players)-1 {
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n")
}

// Equal balances share a rank; names never affect rank movement.
func standingRanks(players []api.TablePlayer) map[string]int {
	ranks := make(map[string]int, len(players))
	for _, player := range players {
		rank := 1
		for _, other := range players {
			if other.Standing > player.Standing {
				rank++
			}
		}
		ranks[player.ID] = rank
	}
	return ranks
}

type tablePlayerLedgerSummary struct {
	withdrawn int
	buyIn     int
	games     int
	change    int
	wins      int
	streak    int
}

func (m Model) tablePlayerLedgerSummaries() map[string]tablePlayerLedgerSummary {
	summaries := make(map[string]tablePlayerLedgerSummary, len(m.table.Players))
	games := append([]api.TableGame(nil), m.table.Games...)
	sort.SliceStable(games, func(i, j int) bool { return games[i].Date < games[j].Date })
	for index, game := range games {
		for _, participant := range game.Participants {
			summary := summaries[participant.PlayerID]
			profit := participant.FinalValue - participant.RequiredEntry
			if index == len(games)-1 {
				summary.change = profit
			}
			switch {
			case profit > 0:
				summary.wins++
				summary.streak = max(summary.streak, 0) + 1
			case profit < 0:
				summary.streak = min(summary.streak, 0) - 1
			default:
				summary.streak = 0
			}
			summary.withdrawn += participant.FinalValue
			summary.buyIn += participant.RequiredEntry
			summary.games++
			summaries[participant.PlayerID] = summary
		}
	}
	return summaries
}

func chartSectionHeadingWithShortcut(title, shortcut string, width int) string {
	prefix := brandStyle.Render("/// " + title)
	right := mutedStyle.Render(shortcut)
	ruleLength := max(width-lipgloss.Width(prefix)-lipgloss.Width(right)-2, 3)
	rule := lipgloss.NewStyle().Foreground(colorIndigo).Render(strings.Repeat("/", ruleLength))
	if shortcut == "" {
		return prefix + " " + rule
	}
	return prefix + " " + rule + " " + right
}

func (m Model) expandedStandingChartHeight() int {
	// Reserve rows for the page header, chart heading, game markers, legend,
	// and the pinned bottom status bar. The chart consumes everything else.
	return max(m.height-9, 8)
}

func displayTablePlayerName(player api.TablePlayer) string {
	username := strings.TrimPrefix(strings.TrimSpace(player.Username), "@")
	if username != "" {
		return "@" + username
	}
	return player.Name
}

func tablePlayerIsHost(player api.TablePlayer, table api.TableSummary) bool {
	return player.UserID != "" && player.UserID == table.HostUserID
}

func (m Model) displayParticipantName(participant api.TableGameParticipant) string {
	name := strings.TrimSpace(participant.PlayerName)
	if m.table == nil {
		return name
	}
	for _, player := range m.table.Players {
		if player.ID == participant.PlayerID && tablePlayerIsHost(player, m.table.Table) {
			return "@" + strings.TrimPrefix(name, "@")
		}
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
