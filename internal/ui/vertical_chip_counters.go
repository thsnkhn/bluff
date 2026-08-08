package ui

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/thsnkhn/bluff/internal/api"
)

// verticalChipCounters is the player-earnings control. One focused row
// represents one denomination; horizontal arrows change its count while
// vertical arrows move between denominations.
type verticalChipCounters struct {
	chips   []api.ChipDenomination
	values  *[]string
	active  int
	width   int
	focused bool
}

func newVerticalChipCounters(chips []api.ChipDenomination, values *[]string) *verticalChipCounters {
	return &verticalChipCounters{chips: chips, values: values, width: 48}
}

func (row *verticalChipCounters) Init() tea.Cmd {
	return row.Focus()
}

func (row *verticalChipCounters) Update(msg tea.Msg) (huh.Model, tea.Cmd) {
	if len(row.chips) == 0 || row.values == nil {
		return row, nil
	}
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return row, nil
	}

	switch keyMsg.String() {
	case "up", "k":
		row.move(-1)
		return row, nil
	case "down", "j":
		row.move(1)
		return row, nil
	case "left", "shift+left":
		step := 1
		if keyMsg.String() == "shift+left" {
			step = 10
		}
		row.adjust(-step)
		return row, nil
	case "right", "shift+right":
		step := 1
		if keyMsg.String() == "shift+right" {
			step = 10
		}
		row.adjust(step)
		return row, nil
	case "enter", "tab":
		return row, func() tea.Msg { return huh.NextField() }
	case "shift+tab":
		return row, func() tea.Msg { return huh.PrevField() }
	}
	return row, nil
}

func (row *verticalChipCounters) move(direction int) {
	row.active = max(min(row.active+direction, len(row.chips)-1), 0)
}

func (row *verticalChipCounters) adjust(delta int) {
	if row.active < 0 || row.active >= len(row.chips) || row.values == nil {
		return
	}
	for len(*row.values) < len(row.chips) {
		*row.values = append(*row.values, "0")
	}
	value, err := strconv.Atoi(strings.TrimSpace((*row.values)[row.active]))
	if err != nil {
		value = 0
	}
	(*row.values)[row.active] = strconv.Itoa(max(value+delta, 0))
}

func (row *verticalChipCounters) View() string {
	if len(row.chips) == 0 || row.values == nil {
		return ""
	}
	width := max(row.width, 24)
	lines := make([]string, 0, len(row.chips))
	for index, chip := range row.chips {
		count := 0
		if index < len(*row.values) {
			count, _ = strconv.Atoi(strings.TrimSpace((*row.values)[index]))
		}
		active := row.focused && index == row.active
		marker := "  "
		denominationStyle := valueStyle
		counterStyle := mutedStyle
		if active {
			marker = "› "
			denominationStyle = lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia)
			counterStyle = denominationStyle
		}
		left := marker + chipSwatch(chip.Color) + " " + denominationStyle.Render(fmt.Sprintf("%s %d", chip.Label, chip.Value))
		right := counterStyle.Render(fmt.Sprintf("- %d +", count))
		gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
		lines = append(lines, left+strings.Repeat(" ", gap)+right)
	}
	return strings.Join(lines, "\n")
}

func (row *verticalChipCounters) Blur() tea.Cmd {
	row.focused = false
	return nil
}

func (row *verticalChipCounters) Focus() tea.Cmd {
	row.focused = true
	return nil
}

func (*verticalChipCounters) Error() error                             { return nil }
func (*verticalChipCounters) Run() error                               { return nil }
func (*verticalChipCounters) RunAccessible(io.Writer, io.Reader) error { return nil }
func (*verticalChipCounters) Skip() bool                               { return false }
func (*verticalChipCounters) Zoom() bool                               { return false }
func (*verticalChipCounters) KeyBinds() []key.Binding                  { return nil }

func (row *verticalChipCounters) WithTheme(huh.Theme) huh.Field    { return row }
func (row *verticalChipCounters) WithKeyMap(*huh.KeyMap) huh.Field { return row }
func (row *verticalChipCounters) WithWidth(width int) huh.Field {
	row.width = width
	return row
}
func (row *verticalChipCounters) WithHeight(int) huh.Field                 { return row }
func (row *verticalChipCounters) WithPosition(huh.FieldPosition) huh.Field { return row }
func (*verticalChipCounters) GetKey() string                               { return "chip-counters" }
func (row *verticalChipCounters) GetValue() any {
	if row.values == nil {
		return []string(nil)
	}
	return append([]string(nil), (*row.values)...)
}
