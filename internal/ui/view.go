package ui

import (
	"fmt"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

func bluffTheme(isDark bool) *huh.Styles {
	theme := huh.ThemeCharm(isDark)
	theme.Focused.Title = theme.Focused.Title.Foreground(colorGold).Bold(true)
	theme.Focused.Description = theme.Focused.Description.Foreground(colorMuted)
	theme.Focused.Base = theme.Focused.Base.BorderForeground(colorGold)
	theme.Focused.TextInput.Cursor = theme.Focused.TextInput.Cursor.Foreground(colorGold)
	theme.Focused.TextInput.Text = theme.Focused.TextInput.Text.Foreground(colorInk)
	theme.Blurred.Title = theme.Blurred.Title.Foreground(colorMuted)
	theme.Help.ShortKey = theme.Help.ShortKey.Foreground(colorGold)
	theme.Help.ShortDesc = theme.Help.ShortDesc.Foreground(colorMuted)
	return theme
}

var (
	colorInk     = lipgloss.Color("#F7F2E8")
	colorMuted   = lipgloss.Color("#918C84")
	colorFelt    = lipgloss.Color("#103D35")
	colorFeltDim = lipgloss.Color("#17352F")
	colorGold    = lipgloss.Color("#E7B85C")
	colorGreen   = lipgloss.Color("#69D3A7")
	colorRed     = lipgloss.Color("#F07878")
	colorPanel   = lipgloss.Color("#142722")
	colorBorder  = lipgloss.Color("#31584E")

	brandStyle = lipgloss.NewStyle().Bold(true).Foreground(colorGold)
	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)
	valueStyle = lipgloss.NewStyle().Bold(true).Foreground(colorInk)
	errorStyle = lipgloss.NewStyle().Foreground(colorRed)
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(colorBorder).
			Background(colorPanel).Padding(1, 2)
)

func (m Model) loadingView() string {
	line := lipgloss.JoinHorizontal(lipgloss.Center, m.spinner.View(), "  ", valueStyle.Render(m.status))
	body := lipgloss.JoinVertical(lipgloss.Center,
		brandStyle.Copy().Render("♠  B L U F F"),
		"",
		line,
		mutedStyle.Render("Private games. One honest ledger."),
	)
	return place(m.width, m.height, body)
}

func (m Model) loginView() string {
	if m.loading {
		return m.loadingView()
	}
	content := []string{
		brandStyle.Copy().Render("♠  B L U F F"),
		mutedStyle.Render("Take your seat"),
		"",
	}
	if m.err != nil {
		content = append(content, errorStyle.Render("! "+friendlyError(m.err)), "")
	}
	content = append(content, m.form.View())
	card := panelStyle.Copy().Width(min(max(m.width-8, 36), 62)).Render(strings.Join(content, "\n"))
	footer := mutedStyle.Render("esc quit")
	return place(m.width, m.height, lipgloss.JoinVertical(lipgloss.Center, card, "", footer))
}

func (m Model) dashboardView() string {
	innerWidth := max(m.width-6, 36)
	header := m.headerView(innerWidth)
	footer := m.footerView(innerWidth)

	var body string
	if innerWidth >= 100 {
		leftWidth := max(30, innerWidth*34/100)
		rightWidth := innerWidth - leftWidth - 2
		left := m.standingsView(leftWidth)
		right := lipgloss.JoinVertical(lipgloss.Left, m.currentGameView(rightWidth), "", m.recentGamesView(rightWidth))
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left,
			m.currentGameView(innerWidth), "",
			m.standingsView(innerWidth), "",
			m.recentGamesView(innerWidth),
		)
	}

	parts := []string{header, "", body}
	if m.err != nil {
		parts = append(parts, "", errorStyle.Render("! "+friendlyError(m.err)))
	}
	parts = append(parts, "", footer)
	return lipgloss.NewStyle().Padding(1, 3).Background(colorFelt).Render(strings.Join(parts, "\n"))
}

func (m Model) headerView(width int) string {
	brand := brandStyle.Copy().Render("♠  B L U F F")
	stats := fmt.Sprintf("%d players  ·  %d games", len(m.bootstrap.Players), len(m.bootstrap.Games))
	identity := lipgloss.NewStyle().Foreground(colorInk).Render("@"+m.user.Username) +
		"  " + lipgloss.NewStyle().Foreground(colorGold).Render(strings.ToUpper(m.user.Role))
	right := lipgloss.JoinVertical(lipgloss.Right, identity, mutedStyle.Render(stats))
	gap := max(width-lipgloss.Width(brand)-lipgloss.Width(right), 1)
	return lipgloss.JoinHorizontal(lipgloss.Top, brand, strings.Repeat(" ", gap), right)
}

func (m Model) standingsView(width int) string {
	lines := []string{sectionTitle("STANDINGS")}
	players := sortedPlayers(m.bootstrap.Players)
	if len(players) == 0 {
		lines = append(lines, "", valueStyle.Render("No players yet"), mutedStyle.Render("Create the first player to open the table."))
	} else {
		for index, player := range players {
			rank := mutedStyle.Render(fmt.Sprintf("%2d", index+1))
			name := valueStyle.Render(truncate(player.Name, max(width-18, 8)))
			amount := standingStyle(player.Standing).Render(signedCredits(player.Standing))
			gap := max(width-8-lipgloss.Width(rank)-lipgloss.Width(name)-lipgloss.Width(amount), 1)
			lines = append(lines, fmt.Sprintf("%s  %s%s%s", rank, name, strings.Repeat(" ", gap), amount))
		}
	}
	return panel(width).Render(strings.Join(lines, "\n"))
}

func (m Model) currentGameView(width int) string {
	lines := []string{sectionTitle("CURRENT GAME")}
	game := m.bootstrap.CurrentGame
	if game == nil {
		lines = append(lines, "", valueStyle.Render("The table is clear"), mutedStyle.Render("No game is currently in progress."))
		return panel(width).Render(strings.Join(lines, "\n"))
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
	return panel(width).Render(strings.Join(lines, "\n"))
}

func (m Model) recentGamesView(width int) string {
	lines := []string{sectionTitle("RECENT GAMES")}
	if len(m.bootstrap.Games) == 0 {
		lines = append(lines, "", valueStyle.Render("No hands in the book"), mutedStyle.Render("Completed games will appear here."))
		return panel(width).Render(strings.Join(lines, "\n"))
	}
	start := max(0, len(m.bootstrap.Games)-5)
	for index := len(m.bootstrap.Games) - 1; index >= start; index-- {
		game := m.bootstrap.Games[index]
		label := valueStyle.Render(truncate(game.Template.Name, max(width-30, 10)))
		meta := mutedStyle.Render(fmt.Sprintf("%s  ·  %d players", game.Date, len(game.Participants)))
		gap := max(width-8-lipgloss.Width(label)-lipgloss.Width(meta)-lipgloss.Width(game.Status)-4, 1)
		lines = append(lines, "", label+strings.Repeat(" ", gap)+statusBadge(game.Status), meta)
	}
	return panel(width).Render(strings.Join(lines, "\n"))
}

func (m Model) footerView(width int) string {
	keys := "r refresh   l logout   q quit"
	version := m.build.Version
	if version == "" {
		version = "dev"
	}
	gap := max(width-len(keys)-len(version), 1)
	loading := ""
	if m.loading {
		loading = m.spinner.View() + " " + m.status + "   "
	}
	return mutedStyle.Render(loading + keys + strings.Repeat(" ", gap) + version)
}

func (m Model) playerName(id string) string {
	for _, player := range m.bootstrap.Players {
		if player.ID == id {
			return player.Name
		}
	}
	return "Unknown player"
}

func panel(width int) lipgloss.Style {
	return panelStyle.Copy().Width(max(width-6, 20))
}

func place(width, height int, content string) string {
	if width <= 0 || height <= 0 {
		return content
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(colorFeltDim)))
}

func sectionTitle(title string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(colorGold).Render(title)
}

func statusBadge(status string) string {
	color := colorMuted
	if status == "active" || status == "correction" {
		color = colorGold
	} else if status == "settled" {
		color = colorGreen
	} else if status == "voided" {
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

func credits(value int) string {
	return fmt.Sprintf("%d cr", value)
}

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
