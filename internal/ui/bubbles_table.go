package ui

import (
	bubblesTable "charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

// newBluffTable keeps Bubbles' table component inside Bluff's existing visual
// language. The component owns a viewport, so long result lists can scroll
// without adding one-off clipping logic to each screen.
func newBluffTable(columns []bubblesTable.Column, rows []bubblesTable.Row, width, height int) bubblesTable.Model {
	table := bubblesTable.New(
		bubblesTable.WithColumns(columns),
		bubblesTable.WithRows(rows),
		bubblesTable.WithWidth(max(width, 1)),
		bubblesTable.WithHeight(max(height, 2)),
		bubblesTable.WithFocused(false),
	)
	styles := bubblesTable.DefaultStyles()
	styles.Header = mutedStyle.Padding(0, 1)
	styles.Cell = lipgloss.NewStyle().Padding(0, 1)
	styles.Selected = lipgloss.NewStyle().Bold(true).Foreground(colorFuchsia)
	table.SetStyles(styles)
	return table
}

// bluffTableRow keeps normal rows on the cream value tone while leaving the
// selected row unstyled so Bubbles can apply the accent style to the full row.
func bluffTableRow(selected bool, values ...string) bubblesTable.Row {
	if selected {
		return bubblesTable.Row(values)
	}
	row := make(bubblesTable.Row, len(values))
	for index, value := range values {
		row[index] = valueStyle.Render(value)
	}
	return row
}

// bluffTableColumns converts the same weighted layout used by the existing
// grids into Bubbles columns. Cell padding is accounted for by
// weightedGridColumns, keeping the table flush with the page content edge.
func bluffTableColumns(width int, titles []string, weights ...int) []bubblesTable.Column {
	columns := make([]bubblesTable.Column, len(titles))
	widths := weightedGridColumns(width, weights...)
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
}

// listTableHeight keeps short lists compact while reserving enough room for
// the page header, action bar, messages, and help bar on longer lists.
func (m Model) listTableHeight(reserve, rows int) int {
	available := max(m.height-reserve, 2)
	return min(available, max(rows+1, 2))
}
