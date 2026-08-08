package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestParseBluffHelpBuildsBindings(t *testing.T) {
	t.Parallel()
	bindings, ok := parseBluffHelp("↑↓ move   enter open   esc back")
	if !ok || len(bindings) != 3 {
		t.Fatalf("bindings = %#v, ok=%v; want three bindings", bindings, ok)
	}
	if bindings[1].Help().Key != "enter" || bindings[1].Help().Desc != "open" {
		t.Fatalf("enter binding = %#v; want enter/open", bindings[1].Help())
	}
}

func TestRenderBluffHelpTruncatesToAvailableWidth(t *testing.T) {
	t.Parallel()
	view := ansi.Strip(renderBluffHelp("↑↓ move   enter open   esc back", 12))
	if !strings.Contains(view, "↑↓") {
		t.Fatalf("truncated help omitted the first key: %q", view)
	}
	if strings.Contains(view, "esc") {
		t.Fatalf("truncated help rendered an item beyond the width: %q", view)
	}
}
