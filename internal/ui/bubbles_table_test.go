package ui

import (
	"fmt"
	"strings"
	"testing"

	bubblesTable "charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/thsnkhn/bluff/internal/api"
)

func TestBluffTableCursorKeepsLongListSelectionVisible(t *testing.T) {
	t.Parallel()
	rows := make([]bubblesTable.Row, 20)
	for index := range rows {
		rows[index] = bubblesTable.Row{fmt.Sprintf("row-%02d", index)}
	}
	table := newBluffTable(bluffTableColumns(40, []string{"ITEM"}, 1), rows, 40, 6)
	setBluffTableCursor(&table, 15)

	if table.Cursor() != 15 {
		t.Fatalf("cursor = %d, want 15", table.Cursor())
	}
	view := bluffTableView(table)
	if !strings.Contains(ansi.Strip(view), "row-15") {
		t.Fatalf("selected row is not visible in long table:\n%s", ansi.Strip(view))
	}
	plain := ansi.Strip(view)
	if !strings.Contains(plain, "┌") || !strings.Contains(plain, "└") || !strings.Contains(plain, "─") {
		t.Fatalf("table does not use the upstream example border style:\n%s", plain)
	}
	for _, line := range strings.Split(view, "\n") {
		if width := ansi.StringWidth(line); width != 40 {
			t.Fatalf("rendered table line width = %d, want 40:\n%s", width, plain)
		}
	}
}

func TestBluffTableSelectedRowOwnsCompleteHighlight(t *testing.T) {
	t.Parallel()
	styled := lipgloss.NewStyle().Foreground(colorFuchsia).Render("colored")
	row := bluffTableRow(true, styled, "plain")
	for index, cell := range row {
		if ansi.Strip(cell) != cell {
			t.Fatalf("selected cell %d retains nested styling that can interrupt the row background: %q", index, cell)
		}
	}
}

func TestSetBluffTableCursorSanitizesActualSelectedRow(t *testing.T) {
	t.Parallel()
	styled := lipgloss.NewStyle().Foreground(colorFuchsia).Render("colored")
	table := newBluffTable(
		bluffTableColumns(40, []string{"ITEM"}, 1),
		[]bubblesTable.Row{{"plain"}, {styled}},
		40,
		4,
	)
	setBluffTableCursor(&table, 1)
	selected := table.SelectedRow()
	if len(selected) != 1 || selected[0] != "colored" {
		t.Fatalf("actual selected row retains nested styling: %#v", selected)
	}
}

func TestTableScrollMessageMovesListSelection(t *testing.T) {
	t.Parallel()
	model := New(fakeAPI{}, fakeStore{}, BuildInfo{})
	model.screen, model.loading = tablesScreen, false
	model.tables = make([]api.TableSummary, 5)

	updated, _ := model.Update(tableScrollMsg{delta: 3})
	got := updated.(Model)
	if got.tableIndex != 3 {
		t.Fatalf("table index after scroll = %d, want 3", got.tableIndex)
	}
}
