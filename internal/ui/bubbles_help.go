package ui

import (
	"strings"

	bubblesHelp "charm.land/bubbles/v2/help"
	bubblesKey "charm.land/bubbles/v2/key"
)

type bluffHelpKeyMap struct {
	bindings []bubblesKey.Binding
}

func (m bluffHelpKeyMap) ShortHelp() []bubblesKey.Binding { return m.bindings }

func (m bluffHelpKeyMap) FullHelp() [][]bubblesKey.Binding { return [][]bubblesKey.Binding{m.bindings} }

// renderBluffHelp adapts Bluff's compact footer strings to Bubbles' help
// component. It keeps the existing three-space separators and lets the
// component truncate gracefully when the connection line leaves less room.
func renderBluffHelp(actions string, width int) string {
	bindings, ok := parseBluffHelp(actions)
	if !ok {
		return mutedStyle.Render(truncate(actions, max(width, 1)))
	}

	help := bubblesHelp.New()
	help.ShortSeparator = "   "
	help.Styles.ShortKey = mutedStyle
	help.Styles.ShortDesc = mutedStyle
	help.Styles.ShortSeparator = mutedStyle
	help.Styles.Ellipsis = mutedStyle
	help.SetWidth(max(width, 1))
	return help.View(bluffHelpKeyMap{bindings: bindings})
}

func parseBluffHelp(actions string) ([]bubblesKey.Binding, bool) {
	actions = strings.TrimSpace(actions)
	if actions == "" {
		return nil, false
	}

	parts := strings.Split(actions, "   ")
	bindings := make([]bubblesKey.Binding, 0, len(parts))
	for _, part := range parts {
		fields := strings.Fields(part)
		if len(fields) < 2 || !isBluffHelpKey(fields[0]) {
			return nil, false
		}
		bindings = append(bindings, bubblesKey.NewBinding(
			bubblesKey.WithKeys(fields[0]),
			bubblesKey.WithHelp(fields[0], strings.Join(fields[1:], " ")),
		))
	}
	return bindings, len(bindings) > 0
}

func isBluffHelpKey(value string) bool {
	if strings.ContainsAny(value, "↑↓←→") || strings.HasPrefix(value, "shift+") {
		return true
	}
	switch value {
	case "tab", "enter", "esc", "backspace", "space", "ctrl+c":
		return true
	default:
		return len([]rune(value)) == 1
	}
}
