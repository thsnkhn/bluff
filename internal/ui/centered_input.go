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
	width       int
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
		Value(value).
		Validate(validate)
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
	lines := strings.Split(input.Input.View(), "\n")
	textLine := len(lines) - 2 // The bottom border is the final line.
	for index, line := range lines {
		if index == textLine {
			plain := ansi.Strip(line)
			leftTrimmed := strings.TrimLeftFunc(plain, unicode.IsSpace)
			left := ansi.StringWidth(plain) - ansi.StringWidth(leftTrimmed)
			visible := ansi.StringWidth(input.placeholder)
			prefix := ""
			if *input.value != "" {
				visible = ansi.StringWidth(*input.value) + 1 // Preserve the cursor cell.
				prefix = " "                                 // Balance the cursor cell so the value itself stays centered.
			}
			visible = min(visible, max(input.width-1, 1))
			lines[index] = centeredLine(prefix+ansi.Cut(line, left, left+visible), input.width)
			continue
		}
		plain := ansi.Strip(line)
		leftTrimmed := strings.TrimLeftFunc(plain, unicode.IsSpace)
		trimmed := strings.TrimRightFunc(leftTrimmed, unicode.IsSpace)
		left := ansi.StringWidth(plain) - ansi.StringWidth(leftTrimmed)
		lines[index] = centeredLine(ansi.Cut(line, left, left+ansi.StringWidth(trimmed)), input.width)
	}
	view := strings.Join(lines, "\n")
	if err := input.Error(); err != nil {
		view += "\n" + centeredLine(errorStyle.Render(err.Error()), input.width)
	}
	return view
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

func (input *centeredInput) WithPosition(position huh.FieldPosition) huh.Field {
	input.Input.WithPosition(position)
	return input
}
