package ui

import (
	"fmt"
	"strings"
	"testing"

	bubblesTable "charm.land/bubbles/v2/table"
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
	if !strings.Contains(ansi.Strip(table.View()), "row-15") {
		t.Fatalf("selected row is not visible in long table:\n%s", ansi.Strip(table.View()))
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
