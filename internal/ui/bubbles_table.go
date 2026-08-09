package ui

import (
	bubblesTable "charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var bluffTableBaseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

// newBluffTable keeps Bubbles' table component inside Bluff's existing visual
// language. The component owns a viewport, so long result lists can scroll
// without adding one-off clipping logic to each screen.
func newBluffTable(columns []bubblesTable.Column, rows []bubblesTable.Row, width, height int) bubblesTable.Model {
	table := bubblesTable.New(
		bubblesTable.WithColumns(columns),
		bubblesTable.WithRows(rows),
		bubblesTable.WithWidth(max(width-2, 1)),
		bubblesTable.WithHeight(max(height, 2)),
		bubblesTable.WithFocused(true),
	)
	styles := bubblesTable.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	styles.Selected = styles.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	table.SetStyles(styles)
	return table
}

// bluffTableRow leaves row styling to Bubbles, matching the upstream example.

func bluffTableRow(selected bool, values ...string) bubblesTable.Row {
	if selected {
		// Nested cell colors reset the selected background partway through a
		// row. The selected treatment owns the complete row, so keep its cell
		// values plain and let Bubbles apply one uninterrupted highlight.
		for index := range values {
			values[index] = ansi.Strip(values[index])
		}
	}
	return bubblesTable.Row(values)
}

func bluffTableView(table bubblesTable.Model) string {
	return bluffTableBaseStyle.Render(table.View())
}

// bluffTableColumns converts the same weighted layout used by the existing
// grids into Bubbles columns. Cell padding is accounted for by
// weightedGridColumns, keeping the table flush with the page content edge.
func bluffTableColumns(width int, titles []string, weights ...int) []bubblesTable.Column {
	columns := make([]bubblesTable.Column, len(titles))
	// Bubbles adds one cell of padding on both sides of every column and Bluff
	// adds the outer border. Reserve both so the final table is exactly width.
	contentWidth := max(width-2-len(titles)*2, len(titles))
	widths := weightedGridColumns(contentWidth, weights...)
	for index, title := range titles {
		columnWidth := 1
		if index < len(widths) {
			columnWidth = max(widths[index], 1)
		}
		columns[index] = bubblesTable.Column{Title: title, Width: columnWidth}
	}
	return columns
}

// tableSelectedRow maps a source index to the row position in a filtered
// table. Games are displayed newest-first, hence the reverse option.
func tableSelectedRow(indices []int, selected int, reverse bool) int {
	for position, index := range indices {
		if index != selected {
			continue
		}
		if reverse {
			return len(indices) - position - 1
		}
		return position
	}
	return 0
}

// setBluffTableCursor also positions Bubbles' internal viewport. SetCursor
// changes the highlight but does not ensure a distant row is visible, so use
// the component's movement API to keep the selected row on screen.
func setBluffTableCursor(table *bubblesTable.Model, row int) {
	rows := table.Rows()
	if len(rows) == 0 {
		return
	}
	row = max(0, min(row, len(rows)-1))
	table.GotoBottom()
	table.MoveUp(len(rows) - 1 - row)

	// The source selection can be absent after filtering while Bubbles still
	// keeps a visible cursor. Sanitize the row at the actual cursor so nested
	// chip, status, or crown colors cannot reset its full-row background.
	selected := append(bubblesTable.Row(nil), rows[row]...)
	for index := range selected {
		selected[index] = ansi.Strip(selected[index])
	}
	rows[row] = selected
	table.SetRows(rows)
}

// listTableHeight keeps short lists compact while reserving enough room for
// the page header, action bar, messages, and help bar on longer lists.
func (m Model) listTableHeight(reserve, rows int) int {
	available := max(m.height-reserve, 2)
	return min(available, max(rows+1, 2))
}
