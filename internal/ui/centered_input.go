package ui

import (
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// centeredInput keeps Huh's input behavior while centering its visible value.
type centeredInput struct {
	*huh.Input
	value       *string
	placeholder string
	prefix      string
	width       int
	inline      bool
	leftAligned bool
}

func newCenteredInput(
	title string,
	description string,
	placeholder string,
	value *string,
	charLimit int,
	password bool,
	validate func(string) error,
) *centeredInput {
	input := huh.NewInput().
		Title(title).
		Description(description).
		Prompt("").
		Placeholder(placeholder).
		Value(value)
	if validate != nil {
		input.Validate(validate)
	}
	if charLimit > 0 {
		input.CharLimit(charLimit)
	}
	if password {
		input.EchoMode(huh.EchoModePassword)
	}
	return &centeredInput{Input: input, value: value, placeholder: placeholder}
}

func (input *centeredInput) Update(msg tea.Msg) (huh.Model, tea.Cmd) {
	_, cmd := input.Input.Update(msg)
	return input, cmd
}

func (input *centeredInput) View() string {
	if input.inline {
		view := input.Input.View()
		if err := input.Error(); err != nil {
			view += "\n" + errorStyle.Render(friendlyError(err))
		}
		return view
	}
	lines := strings.Split(input.Input.View(), "\n")
	textLine := len(lines) - 2 // Centered forms keep the bottom border as the final line.
	if input.leftAligned {
		textLine = len(lines) - 1 // Popup fields use Huh's left focus border only.
	}
	for index, line := range lines {
		if index == textLine {
			if *input.value == "" && input.prefix != "" {
				// Huh reserves a cursor cell when it renders an empty input, which
				// can clip the last placeholder character at narrow widths. Render
				// the complete public identifier ourselves in that state.
				lines[index] = input.alignLine(mutedStyle.Render(input.prefix + input.placeholder))
				continue
			}
			plain := ansi.Strip(line)
			leftTrimmed := strings.TrimLeftFunc(plain, unicode.IsSpace)
			left := ansi.StringWidth(plain) - ansi.StringWidth(leftTrimmed)
			visible := ansi.StringWidth(input.placeholder)
			cursorBalance := ""
			if *input.value != "" {
				visible = ansi.StringWidth(*input.value) + 1 // Preserve the cursor cell.
				if !input.leftAligned {
					cursorBalance = " " // Balance the cursor cell so centered text stays visually stable.
				}
			}
			prefixWidth := ansi.StringWidth(input.prefix)
			visible = min(visible, max(input.width-prefixWidth-ansi.StringWidth(cursorBalance), 1))
			prefix := lipgloss.NewStyle().Foreground(colorFuchsia).Render(input.prefix)
			lines[index] = input.alignLine(cursorBalance + prefix + ansi.Cut(line, left, left+visible))
			continue
		}
		plain := ansi.Strip(line)
		leftTrimmed := strings.TrimLeftFunc(plain, unicode.IsSpace)
		trimmed := strings.TrimRightFunc(leftTrimmed, unicode.IsSpace)
		left := ansi.StringWidth(plain) - ansi.StringWidth(leftTrimmed)
		lines[index] = input.alignLine(ansi.Cut(line, left, left+ansi.StringWidth(trimmed)))
	}
	view := strings.Join(lines, "\n")
	if err := input.Error(); err != nil {
		view += "\n" + input.alignLine(errorStyle.Render(friendlyError(err)))
	}
	return view
}

func (input *centeredInput) alignLine(value string) string {
	if input.leftAligned {
		return lipgloss.NewStyle().Width(input.width).Align(lipgloss.Left).Render(value)
	}
	return centeredLine(value, input.width)
}

func centeredLine(value string, width int) string {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(value)
}

func (input *centeredInput) WithTheme(theme huh.Theme) huh.Field {
	input.Input.WithTheme(theme)
	return input
}

func (input *centeredInput) WithKeyMap(keyMap *huh.KeyMap) huh.Field {
	input.Input.WithKeyMap(keyMap)
	return input
}

func (input *centeredInput) WithWidth(width int) huh.Field {
	input.width = width
	input.Input.WithWidth(width)
	return input
}

func (input *centeredInput) WithHeight(height int) huh.Field {
	input.Input.WithHeight(height)
	return input
}

// WithInline keeps a short label and its value on one line. This is useful for
// compact repeated fields such as chip color and denomination rows.
func (input *centeredInput) WithInline(inline bool) *centeredInput {
	input.Input.Inline(inline)
	input.inline = inline
	return input
}

// WithLeftAlign keeps popup fields readable while the main login forms stay centered.
func (input *centeredInput) WithLeftAlign(left bool) *centeredInput {
	input.leftAligned = left
	return input
}

// WithPrefix renders a visual prefix without including it in the submitted value.
// This keeps canonicalization in the API request while making public identifiers
// clear as they are entered.
func (input *centeredInput) WithPrefix(prefix string) *centeredInput {
	input.prefix = prefix
	return input
}

func (input *centeredInput) WithPosition(position huh.FieldPosition) huh.Field {
	input.Input.WithPosition(position)
	return input
}
