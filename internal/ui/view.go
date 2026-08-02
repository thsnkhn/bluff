package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const homeContentWidth = 64

var (
	colorIndigo  = lipgloss.Color("#7571F9")
	colorFuchsia = lipgloss.Color("#F780E2")
	colorCream   = lipgloss.Color("#FFFDF5")
	colorMuted   = lipgloss.Color("#8B8B92")
	colorGreen   = lipgloss.Color("#02BF87")
	colorRed     = lipgloss.Color("#ED567A")

	brandStyle = lipgloss.NewStyle().Bold(true).Foreground(colorIndigo)
	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)
	valueStyle = lipgloss.NewStyle().Foreground(colorCream)
	errorStyle = lipgloss.NewStyle().Foreground(colorRed)
)

type menuMouseMsg struct {
	index    int
	activate bool
}

type dashboardMouseMsg struct {
	action   string
	activate bool
}

type hitRegion struct {
	x0, x1 int
	y0, y1 int
	value  string
}

func (m Model) loadingView() string {
	line := lipgloss.JoinHorizontal(lipgloss.Center, m.spinner.View(), "  ", valueStyle.Render(m.status))
	body := lipgloss.JoinVertical(lipgloss.Center, suitArt(), "", brandWordmark(), "", line)
	return place(m.width, m.height, body)
}

func (m Model) homeView() string {
	prefix := lipgloss.JoinVertical(lipgloss.Left,
		suitArt(),
		"",
		brandWordmark(),
		mutedStyle.Render("Private games. One honest ledger."),
		"",
		connectionLine(m.connected, m.checkingConnection, m.spinner.View()),
		"",
		brandStyle.Render("What would you like to do?"),
		"",
	)

	items := []struct{ title, detail string }{
		{"Sign in", "Open your private table"},
		{"Check connection", "Verify the Bluff service"},
		{"About Bluff", "Version and product details"},
		{"Quit", "Leave the table"},
	}
	rows := make([]string, 0, len(items))
	for index, item := range items {
		selected := index == m.homeIndex
		if selected {
			background := lipgloss.Color("#1D1B3A")
			selector := lipgloss.NewStyle().Foreground(colorFuchsia).Background(background).Render("› ")
			title := lipgloss.NewStyle().Bold(true).Foreground(colorIndigo).Background(background).Render(item.title)
			gap := max(homeContentWidth-len("› ")-len(item.title)-len(item.detail), 2)
			space := lipgloss.NewStyle().Background(background).Render(strings.Repeat(" ", gap))
			detail := lipgloss.NewStyle().Foreground(colorMuted).Background(background).Render(item.detail)
			rows = append(rows, selector+title+space+detail)
			continue
		}
		left := "  " + valueStyle.Render(item.title)
		gap := max(homeContentWidth-lipgloss.Width(left)-len(item.detail), 2)
		rows = append(rows, left+strings.Repeat(" ", gap)+mutedStyle.Render(item.detail))
	}

	body := prefix + "\n" + strings.Join(rows, "\n") + "\n\n" +
		mutedStyle.Render("↑↓ move   enter select   mouse click   q quit")
	if m.err != nil {
		body += "\n\n" + errorStyle.Render("! "+friendlyError(m.err))
	}
	return place(m.width, m.height, lipgloss.NewStyle().Width(homeContentWidth).Render(body))
}

func (m Model) loginView() string {
	if m.loading {
		return m.loadingView()
	}
	parts := []string{
		suitArt(),
		"",
		brandWordmark(),
		mutedStyle.Render("Sign in to take your seat"),
		"",
	}
	if m.err != nil {
		parts = append(parts, errorStyle.Render("! "+friendlyError(m.err)), "")
	}
	parts = append(parts, m.form.View(), "", mutedStyle.Render("esc back"))
	body := lipgloss.JoinVertical(lipgloss.Center, parts...)
	return place(m.width, m.height, body)
}

func (m Model) aboutView() string {
	version := m.build.Version
	if version == "" {
		version = "dev"
	}
	body := lipgloss.JoinVertical(lipgloss.Center,
		suitArt(),
		"",
		brandWordmark(),
		mutedStyle.Render("Private games. One honest ledger."),
		"",
		valueStyle.Render("A shared bankroll for your poker table."),
		mutedStyle.Render("Credentials stay in your system keychain."),
		"",
		brandStyle.Render(version),
		"",
		mutedStyle.Render("enter or esc back"),
	)
	return place(m.width, m.height, body)
}

func (m Model) dashboardView() string {
	innerWidth := max(m.width-6, 36)
	header := m.headerView(innerWidth)

	var body string
	if innerWidth >= 100 {
		leftWidth := max(30, innerWidth*34/100)
		rightWidth := innerWidth - leftWidth - 3
		left := m.standingsView(leftWidth)
		right := lipgloss.JoinVertical(lipgloss.Left, m.currentGameView(rightWidth), "", m.recentGamesView(rightWidth))
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, "   ", right)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left,
			m.currentGameView(innerWidth), "",
			m.standingsView(innerWidth), "",
			m.recentGamesView(innerWidth),
		)
	}

	mainParts := []string{header, "", body}
	if m.err != nil {
		mainParts = append(mainParts, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	main := lipgloss.NewStyle().Padding(1, 3).Render(strings.Join(mainParts, "\n"))
	footer := lipgloss.NewStyle().PaddingLeft(3).Render(m.footerView(innerWidth))
	gap := max(m.height-lipgloss.Height(main)-lipgloss.Height(footer), 0)
	return main + strings.Repeat("\n", gap) + footer
}

func (m Model) headerView(width int) string {
	title := sectionHeading("Bluff Bankroll", width)
	identity := valueStyle.Render("@"+m.user.Username) + "  " +
		lipgloss.NewStyle().Foreground(colorFuchsia).Render(strings.ToUpper(m.user.Role))
	stats := mutedStyle.Render(fmt.Sprintf("%d players  ·  %d games", len(m.bootstrap.Players), len(m.bootstrap.Games)))
	return title + "\n" + identity + "  " + stats
}

func (m Model) standingsView(width int) string {
	lines := []string{sectionHeading("Standings", width)}
	players := sortedPlayers(m.bootstrap.Players)
	if len(players) == 0 {
		lines = append(lines, "", valueStyle.Render("No players yet"), mutedStyle.Render("Create the first player to open the table."))
	} else {
		for index, player := range players {
			rank := mutedStyle.Render(fmt.Sprintf("%2d", index+1))
			name := valueStyle.Render(truncate(player.Name, max(width-18, 8)))
			amount := standingStyle(player.Standing).Render(signedCredits(player.Standing))
			gap := max(width-lipgloss.Width(rank)-lipgloss.Width(name)-lipgloss.Width(amount)-2, 1)
			lines = append(lines, rank+"  "+name+strings.Repeat(" ", gap)+amount)
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) currentGameView(width int) string {
	lines := []string{sectionHeading("Current Game", width)}
	game := m.bootstrap.CurrentGame
	if game == nil {
		lines = append(lines, "", valueStyle.Render("The table is clear"), mutedStyle.Render("No game is currently in progress."))
		return strings.Join(lines, "\n")
	}
	names := make([]string, 0, len(game.Participants))
	for _, participant := range game.Participants {
		names = append(names, m.playerName(participant.PlayerID))
	}
	lines = append(lines,
		"",
		valueStyle.Render(game.Template.Name)+"  "+statusBadge(game.Status),
		mutedStyle.Render(game.Date+"  ·  "+fmt.Sprintf("%d players", len(game.Participants))),
		"",
		strings.Join(names, mutedStyle.Render("  •  ")),
		"",
		mutedStyle.Render("Table value")+"  "+valueStyle.Render(credits(game.ExpectedTableValue)),
	)
	return strings.Join(lines, "\n")
}

func (m Model) recentGamesView(width int) string {
	lines := []string{sectionHeading("Recent Games", width)}
	if len(m.bootstrap.Games) == 0 {
		lines = append(lines, "", valueStyle.Render("No hands in the book"), mutedStyle.Render("Completed games will appear here."))
		return strings.Join(lines, "\n")
	}
	start := max(0, len(m.bootstrap.Games)-5)
	for index := len(m.bootstrap.Games) - 1; index >= start; index-- {
		game := m.bootstrap.Games[index]
		label := valueStyle.Render(truncate(game.Template.Name, max(width-30, 10)))
		meta := mutedStyle.Render(fmt.Sprintf("%s  ·  %d players", game.Date, len(game.Participants)))
		gap := max(width-lipgloss.Width(label)-lipgloss.Width(meta)-lipgloss.Width(game.Status)-4, 1)
		lines = append(lines, "", label+strings.Repeat(" ", gap)+statusBadge(game.Status), meta)
	}
	return strings.Join(lines, "\n")
}

func (m Model) footerView(width int) string {
	version := m.build.Version
	if version == "" {
		version = "dev"
	}
	actions := lipgloss.NewStyle().Foreground(colorFuchsia).Render("[r] refresh") + "   " +
		lipgloss.NewStyle().Foreground(colorFuchsia).Render("[l] logout") + "   " +
		lipgloss.NewStyle().Foreground(colorFuchsia).Render("[q] quit")
	if m.loading {
		actions = m.spinner.View() + " " + valueStyle.Render(m.status)
	}
	gap := max(width-lipgloss.Width(actions)-len(version), 1)
	return actions + strings.Repeat(" ", gap) + mutedStyle.Render(version)
}

func suitArt() string {
	red := lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia)
	black := lipgloss.NewStyle().Bold(true).Foreground(colorIndigo)
	diamond := strings.Join([]string{
		"   /\\   ",
		"  /  \\  ",
		" <    > ",
		"  \\  /  ",
		"   \\/   ",
	}, "\n")
	club := strings.Join([]string{
		"   ()   ",
		" ()  () ",
		"   ()   ",
		"   /\\   ",
		"  /__\\  ",
	}, "\n")
	spade := strings.Join([]string{
		"   /\\   ",
		"  /  \\  ",
		" (    ) ",
		"  \\()/  ",
		"  /__\\  ",
	}, "\n")
	heart := strings.Join([]string{
		" ()  () ",
		"(      )",
		" \\    / ",
		"  \\  /  ",
		"   \\/   ",
	}, "\n")
	return lipgloss.JoinHorizontal(lipgloss.Center,
		red.Render(diamond), "  ", black.Render(club), "  ",
		red.Render(spade), "  ", black.Render(heart),
	)
}

func brandWordmark() string {
	return brandStyle.Render("B  L  U  F  F")
}

func connectionLine(connected, checking bool, spinnerView string) string {
	if checking {
		return spinnerView + " " + mutedStyle.Render("Checking connection")
	}
	if connected {
		return lipgloss.NewStyle().Foreground(colorGreen).Render("● Connected") + mutedStyle.Render("  api.bluff.thsnkhn.com")
	}
	return errorStyle.Render("● Offline") + mutedStyle.Render("  check your connection")
}

func sectionHeading(title string, width int) string {
	styledTitle := brandStyle.Render(title)
	ruleLength := max(width-lipgloss.Width(styledTitle)-1, 4)
	rule := lipgloss.NewStyle().Foreground(colorIndigo).Render(strings.Repeat("/", ruleLength))
	return styledTitle + " " + rule
}

func (m Model) mouseHandler() func(tea.MouseMsg) tea.Cmd {
	return func(msg tea.MouseMsg) tea.Cmd {
		mouse := msg.Mouse()
		activate := false
		if _, ok := msg.(tea.MouseClickMsg); ok && mouse.Button == tea.MouseLeft {
			activate = true
		}
		switch m.screen {
		case homeScreen:
			for index, region := range m.homeHitRegions() {
				if inRegion(mouse.X, mouse.Y, region) {
					return func() tea.Msg { return menuMouseMsg{index: index, activate: activate} }
				}
			}
		case dashboardScreen:
			for _, region := range m.dashboardHitRegions() {
				if inRegion(mouse.X, mouse.Y, region) {
					return func() tea.Msg { return dashboardMouseMsg{action: region.value, activate: activate} }
				}
			}
		}
		return nil
	}
}

func (m Model) homeHitRegions() []hitRegion {
	prefix := lipgloss.JoinVertical(lipgloss.Left,
		suitArt(), "", brandWordmark(), mutedStyle.Render("Private games. One honest ledger."), "",
		connectionLine(m.connected, m.checkingConnection, m.spinner.View()), "",
		brandStyle.Render("What would you like to do?"), "",
	)
	bodyHeight := lipgloss.Height(prefix) + homeItemCount + 2 + 1
	if m.err != nil {
		bodyHeight += 2
	}
	x := max((m.width-homeContentWidth)/2, 0)
	y := max((m.height-bodyHeight)/2, 0) + lipgloss.Height(prefix)
	regions := make([]hitRegion, homeItemCount)
	for index := range regions {
		regions[index] = hitRegion{x0: x, x1: x + homeContentWidth, y0: y + index, y1: y + index, value: fmt.Sprint(index)}
	}
	return regions
}

func (m Model) dashboardHitRegions() []hitRegion {
	y := max(m.height-1, 0)
	return []hitRegion{
		{x0: 3, x1: 13, y0: y, y1: y, value: "refresh"},
		{x0: 16, x1: 26, y0: y, y1: y, value: "logout"},
		{x0: 29, x1: 37, y0: y, y1: y, value: "quit"},
	}
}

func inRegion(x, y int, region hitRegion) bool {
	return x >= region.x0 && x <= region.x1 && y >= region.y0 && y <= region.y1
}

func (m Model) playerName(id string) string {
	for _, player := range m.bootstrap.Players {
		if player.ID == id {
			return player.Name
		}
	}
	return "Unknown player"
}

func place(width, height int, content string) string {
	if width <= 0 || height <= 0 {
		return content
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

func statusBadge(status string) string {
	color := colorMuted
	switch status {
	case "active", "correction":
		color = colorFuchsia
	case "settled":
		color = colorGreen
	case "voided":
		color = colorRed
	}
	return lipgloss.NewStyle().Bold(true).Foreground(color).Render(strings.ToUpper(status))
}

func standingStyle(value int) lipgloss.Style {
	color := colorMuted
	if value > 0 {
		color = colorGreen
	} else if value < 0 {
		color = colorRed
	}
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

func signedCredits(value int) string {
	if value > 0 {
		return "+" + credits(value)
	}
	return credits(value)
}

func credits(value int) string { return fmt.Sprintf("%d cr", value) }

func truncate(value string, limit int) string {
	if limit < 2 || len([]rune(value)) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit-1]) + "…"
}

func friendlyError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if len(message) > 120 {
		return message[:119] + "…"
	}
	return message
}
