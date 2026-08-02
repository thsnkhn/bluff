package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	homeContentWidth = 76
	homeMenuWidth    = 32
)

const blockLogo = ` ▒▒▒          ▒▒▒                  ▒▒▒▒▒    ▒▒▒▒▒▒
▒▒▒▒         ▒▒▒▒                ▒▒▒   ▒  ▒▒▒▒  ▒▒
▒▒▒▒         ▒▒▒▒               ▒▒▒▒      ▒▒▒▒
▒▒▒▒▒▒▒▒▒▒▒  ▒▒▒▒ ▒▒▒▒   ▒▒▒▒ ▒▒▒▒▒▒▒▒▒ ▒▒▒▒▒▒▒▒▒
▒▒▒▒   ▒▒▒▒▒ ▒▒▒▒ ▒▒▒▒   ▒▒▒▒   ▒▒▒▒      ▒▒▒▒
▒▒▒▒    ▒▒▒▒ ▒▒▒▒ ▒▒▒▒   ▒▒▒▒   ▒▒▒▒      ▒▒▒▒
▒▒▒▒    ▒▒▒▒ ▒▒▒▒ ▒▒▒▒   ▒▒▒▒   ▒▒▒▒      ▒▒▒▒
▒▒▒▒    ▒▒▒  ▒▒▒▒ ▒▒▒▒   ▒▒▒▒   ▒▒▒▒      ▒▒▒▒
 ▒▒▒▒▒▒▒▒    ▒▒▒▒  ▒▒▒▒▒▒▒▒▒▒   ▒▒▒▒      ▒▒▒▒`

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

type appMenuMouseMsg struct {
	index    int
	activate bool
}

type usersMouseMsg struct {
	action    string
	userIndex int
	activate  bool
}

type hitRegion struct {
	x0, x1 int
	y0, y1 int
	value  string
}

func (m Model) loadingView() string {
	line := lipgloss.JoinHorizontal(lipgloss.Center, m.spinner.View(), "  ", valueStyle.Render(m.status))
	body := lipgloss.JoinVertical(lipgloss.Center, brandLogo(m.width), "", line)
	return m.screenView(body, "please wait")
}

func (m Model) homeView() string {
	prefix := m.homePrefix()
	items := []string{"Sign in", "Have an invite code?", "Check connection", "About Bluff", "Quit"}
	rows := make([]string, 0, len(items))
	for index, title := range items {
		selected := index == m.homeIndex
		if selected {
			rows = append(rows, lipgloss.NewStyle().
				Width(homeMenuWidth).
				Align(lipgloss.Center).
				Bold(true).
				Foreground(colorFuchsia).
				Render("› "+title+" ‹"))
			continue
		}
		rows = append(rows, lipgloss.NewStyle().Width(homeMenuWidth).Align(lipgloss.Center).Foreground(colorCream).Render(title))
	}

	parts := []string{
		prefix,
		lipgloss.JoinVertical(lipgloss.Center, rows...),
	}
	if m.err != nil {
		parts = append(parts, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	body := lipgloss.JoinVertical(lipgloss.Center, parts...)
	return m.screenView(lipgloss.NewStyle().Width(m.homeWidth()).Align(lipgloss.Center).Render(body),
		"↑↓ move   enter select   mouse click   q quit")
}

func (m Model) homePrefix() string {
	return lipgloss.JoinVertical(lipgloss.Center,
		brandLogo(m.width),
		"",
		"",
		"",
	)
}

func (m Model) loginView() string {
	if m.loading {
		return m.loadingView()
	}
	parts := []string{
		brandLogo(m.width),
		"",
		"",
	}
	if m.err != nil {
		parts = append(parts, errorStyle.Render("! "+friendlyError(m.err)), "")
	}
	parts = append(parts, m.form.View())
	body := lipgloss.JoinVertical(lipgloss.Center, parts...)
	return m.screenView(body, "tab next   enter continue   esc back")
}

func (m Model) inviteCodeView() string {
	if m.loading {
		return m.loadingView()
	}
	parts := []string{brandLogo(m.width), "", "", brandStyle.Render("Join the table"), ""}
	if m.err != nil {
		parts = append(parts, errorStyle.Render("! "+friendlyError(m.err)), "")
	}
	parts = append(parts, m.form.View())
	body := lipgloss.JoinVertical(lipgloss.Center, parts...)
	return m.screenView(body, "enter continue   esc back")
}

func (m Model) inviteAccountView() string {
	if m.loading {
		return m.loadingView()
	}
	parts := []string{
		brandLogo(m.width), "", "",
		brandStyle.Render("Choose your login"),
		mutedStyle.Render("Invite ") + valueStyle.Render(strings.ToUpper(m.invite.code)),
		"",
	}
	if m.err != nil {
		parts = append(parts, errorStyle.Render("! "+friendlyError(m.err)), "")
	}
	parts = append(parts, m.form.View())
	body := lipgloss.JoinVertical(lipgloss.Center, parts...)
	return m.screenView(body, "tab next   enter create account   esc back")
}

func (m Model) aboutView() string {
	version := m.build.Version
	if version == "" {
		version = "dev"
	}
	body := lipgloss.JoinVertical(lipgloss.Center,
		brandLogo(m.width),
		"",
		"",
		valueStyle.Render("Track games. Settle balances. Keep the table honest."),
		mutedStyle.Render("Fast, accessible, and easy for everyone at the table."),
		"",
		mutedStyle.Render("Made with love by ")+lipgloss.NewStyle().Foreground(colorFuchsia).Render("@thsnkhn"),
		"",
		brandStyle.Render(version),
	)
	return m.screenView(body, "enter / esc / q back")
}

func (m Model) appMenuView() string {
	items := m.appMenuItems()
	rows := make([]string, 0, len(items))
	for index, item := range items {
		style := valueStyle
		label := item.title
		if !item.enabled {
			style = mutedStyle
			label += "  soon"
		} else if index == m.appIndex {
			style = lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia)
			label = "› " + label + " ‹"
		}
		rows = append(rows, style.Width(homeMenuWidth).Align(lipgloss.Center).Render(label))
	}

	parts := []string{
		brandLogo(m.width), "", "",
		m.identityLine(),
		"",
		brandStyle.Render("Choose your next move."),
		"",
		lipgloss.JoinVertical(lipgloss.Center, rows...),
	}
	if m.err != nil {
		parts = append(parts, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	body := lipgloss.NewStyle().Width(m.homeWidth()).Align(lipgloss.Center).
		Render(lipgloss.JoinVertical(lipgloss.Center, parts...))
	return m.screenView(body, "↑↓ move   enter select   mouse click   q quit")
}

func (m Model) usersView() string {
	if m.loading {
		return m.pageView(m.loadingViewBody(), usersFooter())
	}
	pageWidth := max(m.width-4, 44)
	contentWidth := min(pageWidth, 96)
	parts := []string{
		pageHeader(pageWidth, "users"),
		"",
		actionBar(usersActionBarItems(), m.usersActionHover),
		"",
		m.userList(contentWidth),
	}
	if m.notice != "" {
		parts = append(parts, "", lipgloss.NewStyle().Foreground(colorGreen).Render("✓ "+m.notice))
	}
	if m.err != nil {
		parts = append(parts, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	body := lipgloss.NewStyle().Width(pageWidth).Render(strings.Join(parts, "\n"))
	return m.pageView(body, usersFooter())
}

func pageHeader(width int, breadcrumbs ...string) string {
	trail := ""
	if len(breadcrumbs) > 0 {
		trail = mutedStyle.Render(" / ") + valueStyle.Render(strings.Join(breadcrumbs, " / "))
	}
	left := lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia).Render("~bluff") + trail
	ruleWidth := max(width-lipgloss.Width(left)-1, 4)
	rule := lipgloss.NewStyle().Foreground(colorIndigo).Render(strings.Repeat("/", ruleWidth))
	return left + " " + rule
}

func (m Model) identityLine() string {
	return valueStyle.Render("@"+m.user.Username) + "  " +
		lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia).Render(strings.ToUpper(m.user.Role))
}

func (m Model) userList(width int) string {
	if len(m.users) == 0 {
		return lipgloss.JoinVertical(lipgloss.Center,
			valueStyle.Render("No users yet"),
			mutedStyle.Render("Create an invite code to welcome someone."),
		)
	}
	usernameWidth := max(width-25, 16)
	lines := []string{mutedStyle.Render(fmt.Sprintf("%-4s %-*s %s", "", usernameWidth, "USERNAME", "ROLE"))}
	for index, user := range m.users {
		marker := "  "
		style := valueStyle
		if index == m.usersIndex {
			marker = "› "
			style = lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia)
		}
		role := strings.ToUpper(user.Role)
		line := fmt.Sprintf("%-4s %-*s %s", marker, usernameWidth, "@"+truncate(user.Username, usernameWidth-1), role)
		lines = append(lines, style.Render(line))
	}
	lines = append(lines, "", mutedStyle.Render(fmt.Sprintf("%d users", len(m.users))))
	return strings.Join(lines, "\n")
}

func usersFooter() string {
	return "↑↓ move   c invite   r refresh   esc back   mouse click"
}

func (m Model) loadingViewBody() string {
	width := max(m.width-4, 44)
	return lipgloss.JoinVertical(lipgloss.Left,
		pageHeader(width, "users"),
		"", m.spinner.View()+"  "+valueStyle.Render(m.status),
	)
}

func pinnedView(width, height int, content, footer string) string {
	if width <= 0 || height <= 1 {
		return content + "\n" + footer
	}
	body := lipgloss.Place(width, height-1, lipgloss.Center, lipgloss.Center, content)
	bottom := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(footer)
	return body + "\n" + bottom
}

func (m Model) screenView(content, actions string) string {
	return pinnedView(m.width, m.height, content, m.helpBar(actions))
}

func (m Model) pageView(content, actions string) string {
	return pinnedTopView(m.width, m.height, content, m.helpBar(actions))
}

func pinnedTopView(width, height int, content, footer string) string {
	if width <= 0 || height <= 1 {
		return content + "\n" + footer
	}
	main := lipgloss.NewStyle().MarginTop(1).MarginLeft(2).Render(content)
	body := lipgloss.Place(width, height-1, lipgloss.Left, lipgloss.Top, main)
	bottom := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(footer)
	return body + "\n" + bottom
}

func (m Model) helpBar(actions string) string {
	connection := m.footerIdentity(connectionLine(m.connected, m.checkingConnection, m.spinner.View()))
	styledActions := mutedStyle.Render(actions)
	available := max(m.width-2, 1)
	gap := available - lipgloss.Width(connection) - lipgloss.Width(styledActions)
	if gap >= 3 {
		return lipgloss.NewStyle().Width(available).Render(connection + strings.Repeat(" ", gap) + styledActions)
	}
	compactConnection := m.footerIdentity(compactConnectionLine(m.connected, m.checkingConnection, m.spinner.View()))
	gap = available - lipgloss.Width(compactConnection) - lipgloss.Width(styledActions)
	if gap < 2 {
		remaining := available - lipgloss.Width(compactConnection) - 2
		if remaining <= 0 {
			return compactConnection
		}
		return compactConnection + "  " + mutedStyle.Render(truncate(actions, remaining))
	}
	return compactConnection + strings.Repeat(" ", gap) + styledActions
}

func (m Model) footerIdentity(connection string) string {
	if m.user.Username == "" {
		return connection
	}
	separator := mutedStyle.Render("  ·  ")
	return connection + separator + m.identityLine()
}

func (m Model) dashboardView() string {
	innerWidth := max(m.width-4, 36)
	header := pageHeader(innerWidth, "dashboard")

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

	stats := mutedStyle.Render(fmt.Sprintf("%d players  ·  %d games", len(m.bootstrap.Players), len(m.bootstrap.Games)))
	mainParts := []string{header, stats, "", body}
	if m.err != nil {
		mainParts = append(mainParts, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	main := lipgloss.NewStyle().Width(innerWidth).Render(strings.Join(mainParts, "\n"))
	return m.pageView(main, m.dashboardHelp())
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

func (m Model) dashboardHelp() string {
	version := m.build.Version
	if version == "" {
		version = "dev"
	}
	if m.loading {
		return m.status
	}
	return "r refresh   l logout   q quit   " + version
}

func brandLogo(width int) string {
	if width > 0 && width < lipgloss.Width(blockLogo)+4 {
		return brandStyle.Render("▓▓  B L U F F  ▓▓")
	}
	return gradientLogo(blockLogo)
}

func gradientLogo(logo string) string {
	lines := strings.Split(logo, "\n")
	logoWidth := 0
	for _, line := range lines {
		logoWidth = max(logoWidth, lipgloss.Width(line))
	}

	start := [3]int{247, 128, 226}
	end := [3]int{117, 113, 249}
	last := max(len(lines)-1, 1)
	for index, line := range lines {
		color := lipgloss.Color(fmt.Sprintf("#%02X%02X%02X",
			start[0]+(end[0]-start[0])*index/last,
			start[1]+(end[1]-start[1])*index/last,
			start[2]+(end[2]-start[2])*index/last,
		))
		padded := line + strings.Repeat(" ", logoWidth-lipgloss.Width(line))
		lines[index] = lipgloss.NewStyle().Bold(true).Foreground(color).Render(padded)
	}
	return strings.Join(lines, "\n")
}

func connectionLine(connected, checking bool, spinnerView string) string {
	if checking {
		return spinnerView + " " + mutedStyle.Render("Checking connection")
	}
	if connected {
		return lipgloss.NewStyle().Foreground(colorGreen).Render("● Connected")
	}
	return errorStyle.Render("● Offline") + mutedStyle.Render("  check your connection")
}

func compactConnectionLine(connected, checking bool, spinnerView string) string {
	if checking {
		return spinnerView + " " + mutedStyle.Render("Checking")
	}
	if connected {
		return lipgloss.NewStyle().Foreground(colorGreen).Render("● Connected")
	}
	return errorStyle.Render("● Offline")
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
		case appMenuScreen:
			for index, region := range m.appMenuHitRegions() {
				if inRegion(mouse.X, mouse.Y, region) {
					return func() tea.Msg { return appMenuMouseMsg{index: index, activate: activate} }
				}
			}
		case usersScreen:
			for _, region := range m.usersHitRegions() {
				if inRegion(mouse.X, mouse.Y, region) {
					userIndex := -1
					if strings.HasPrefix(region.value, "user:") {
						_, _ = fmt.Sscanf(region.value, "user:%d", &userIndex)
					}
					return func() tea.Msg {
						return usersMouseMsg{action: region.value, userIndex: userIndex, activate: activate}
					}
				}
			}
			return func() tea.Msg { return usersMouseMsg{userIndex: -1} }
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
	prefix := m.homePrefix()
	bodyHeight := lipgloss.Height(prefix) + homeItemCount
	if m.err != nil {
		bodyHeight += 2
	}
	x := max((m.width-homeMenuWidth)/2, 0)
	y := max((m.height-1-bodyHeight)/2, 0) + lipgloss.Height(prefix)
	regions := make([]hitRegion, homeItemCount)
	for index := range regions {
		regions[index] = hitRegion{x0: x, x1: x + homeMenuWidth, y0: y + index, y1: y + index, value: fmt.Sprint(index)}
	}
	return regions
}

func (m Model) appMenuHitRegions() []hitRegion {
	items := m.appMenuItems()
	if len(items) == 0 {
		return nil
	}
	partsBeforeMenu := []string{
		brandLogo(m.width), "", "",
		m.identityLine(), "",
		brandStyle.Render("Choose your next move."), "",
	}
	prefixHeight := lipgloss.Height(lipgloss.JoinVertical(lipgloss.Center, partsBeforeMenu...))
	bodyHeight := prefixHeight + len(items)
	if m.err != nil {
		bodyHeight += 2
	}
	x := max((m.width-homeMenuWidth)/2, 0)
	y := max((m.height-1-bodyHeight)/2, 0) + prefixHeight
	regions := make([]hitRegion, len(items))
	for index := range items {
		regions[index] = hitRegion{x0: x, x1: x + homeMenuWidth, y0: y + index, y1: y + index, value: string(items[index].action)}
	}
	return regions
}

func (m Model) usersHitRegions() []hitRegion {
	if m.loading {
		return nil
	}
	pageWidth := max(m.width-4, 44)
	width := min(pageWidth, 96)
	x, y := 2, 1
	shortcutY := y + lipgloss.Height(pageHeader(pageWidth, "users")) + 1
	items := usersActionBarItems()
	shortcutHeight := lipgloss.Height(actionBar(items, m.usersActionHover))
	regions := actionBarHitRegions(x, shortcutY, items)
	listY := shortcutY + shortcutHeight + 1
	for index := range m.users {
		regions = append(regions, hitRegion{x0: x, x1: x + width, y0: listY + 1 + index, y1: listY + 1 + index, value: fmt.Sprintf("user:%d", index)})
	}
	return regions
}

func (m Model) homeWidth() int {
	if m.width <= 0 {
		return homeContentWidth
	}
	return min(homeContentWidth, max(m.width-2, homeMenuWidth))
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
