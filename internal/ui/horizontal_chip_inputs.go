package ui

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

type chipInputSpec struct {
	color       string
	placeholder string
	value       *string
	validate    func(string) error
}

// horizontalChipInputs composes Huh inputs into a compact, reusable chip grid.
// It keeps Huh validation and keyboard behavior while presenting denominations
// as color swatches with adjacent value fields.
type horizontalChipInputs struct {
	fields  []*centeredInput
	colors  []string
	active  int
	width   int
	focused bool
}

func newHorizontalChipInputs(specs []chipInputSpec) *horizontalChipInputs {
	row := &horizontalChipInputs{width: 48}
	for _, spec := range specs {
		row.colors = append(row.colors, spec.color)
		row.fields = append(row.fields, newCenteredInput("", "", spec.placeholder, spec.value, 10, false, spec.validate).WithLeftAlign(true))
	}
	return row
}

func (row *horizontalChipInputs) Init() tea.Cmd {
	if len(row.fields) == 0 {
		return nil
	}
	return row.fields[row.active].Init()
}

func (row *horizontalChipInputs) Update(msg tea.Msg) (huh.Model, tea.Cmd) {
	if len(row.fields) == 0 {
		return row, nil
	}
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "tab", "right", "enter":
			if err := row.fields[row.active].Error(); err != nil {
				return row, nil
			}
			if row.active == len(row.fields)-1 {
				if keyMsg.String() == "enter" || keyMsg.String() == "tab" {
					return row, func() tea.Msg { return huh.NextField() }
				}
				return row, nil
			}
			return row, row.move(1)
		case "shift+tab", "left":
			if row.active == 0 {
				return row, func() tea.Msg { return huh.PrevField() }
			}
			return row, row.move(-1)
		}
	}
	_, cmd := row.fields[row.active].Update(msg)
	return row, cmd
}

func (row *horizontalChipInputs) move(direction int) tea.Cmd {
	_ = row.fields[row.active].Blur()
	row.active += direction
	if row.active < 0 {
		row.active = 0
	}
	if row.active >= len(row.fields) {
		row.active = len(row.fields) - 1
	}
	return row.fields[row.active].Focus()
}

func (row *horizontalChipInputs) View() string {
	if len(row.fields) == 0 {
		return ""
	}
	// Four columns keep the swatches and value fields legible in the format
	// popup while still giving the final row room for the remaining colors.
	columns := min(len(row.fields), 4)
	cellWidth := max((row.width-(columns-1)*2)/columns, 7)
	lines := make([]string, 0, (len(row.fields)+columns-1)/columns)
	for start := 0; start < len(row.fields); start += columns {
		end := min(start+columns, len(row.fields))
		cells := make([]string, 0, end-start)
		for index := start; index < end; index++ {
			value := strings.TrimSpace(*row.fields[index].value)
			display := value
			textColor := colorCream
			if display == "" {
				display = row.fields[index].placeholder
				textColor = colorMuted
			}
			if row.focused && index == row.active {
				display += "▌"
			}
			inputWidth := max(cellWidth-3, 3)
			inputStyle := lipgloss.NewStyle().Width(inputWidth).Foreground(textColor)
			if row.focused && index == row.active {
				inputStyle = inputStyle.BorderLeft(true).BorderForeground(colorFuchsia)
			}
			input := inputStyle.Render(truncate(display, inputWidth))
			cells = append(cells, chipSwatch(row.colors[index])+" "+input)
		}
		lines = append(lines, strings.Join(cells, "  "))
	}
	if err := row.Error(); err != nil {
		lines = append(lines, errorStyle.Render("! "+friendlyError(err)))
	}
	return strings.Join(lines, "\n")
}

func (row *horizontalChipInputs) Blur() tea.Cmd {
	row.focused = false
	if len(row.fields) == 0 {
		return nil
	}
	return row.fields[row.active].Blur()
}

func (row *horizontalChipInputs) Focus() tea.Cmd {
	row.focused = true
	if len(row.fields) == 0 {
		return nil
	}
	return row.fields[row.active].Focus()
}

func (row *horizontalChipInputs) Error() error {
	for _, field := range row.fields {
		if err := field.Error(); err != nil {
			return err
		}
	}
	return nil
}

func (*horizontalChipInputs) Run() error { return nil }

func (row *horizontalChipInputs) RunAccessible(w io.Writer, r io.Reader) error {
	if len(row.fields) == 0 {
		return nil
	}
	return row.fields[row.active].RunAccessible(w, r)
}

func (*horizontalChipInputs) Skip() bool { return false }
func (*horizontalChipInputs) Zoom() bool { return false }

func (row *horizontalChipInputs) KeyBinds() []key.Binding {
	if len(row.fields) == 0 {
		return nil
	}
	return row.fields[row.active].KeyBinds()
}

func (row *horizontalChipInputs) WithTheme(theme huh.Theme) huh.Field {
	for _, field := range row.fields {
		field.WithTheme(theme)
	}
	return row
}

func (row *horizontalChipInputs) WithKeyMap(keyMap *huh.KeyMap) huh.Field {
	for _, field := range row.fields {
		field.WithKeyMap(keyMap)
	}
	return row
}

func (row *horizontalChipInputs) WithWidth(width int) huh.Field {
	row.width = width
	return row
}

func (row *horizontalChipInputs) WithHeight(height int) huh.Field { return row }

func (row *horizontalChipInputs) WithPosition(position huh.FieldPosition) huh.Field {
	for _, field := range row.fields {
		field.WithPosition(position)
	}
	return row
}

func (*horizontalChipInputs) GetKey() string { return "chip-values" }

func (row *horizontalChipInputs) GetValue() any {
	values := make([]string, 0, len(row.fields))
	for _, field := range row.fields {
		values = append(values, *field.value)
	}
	return values
}
