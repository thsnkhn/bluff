package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const actionBarGap = 4

type actionBarItem struct {
	key    string
	label  string
	action string
	accent bool
}

func actionBar(items []actionBarItem, hoveredAction string) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		label := item.key + "  " + item.label
		style := mutedStyle
		if item.action == hoveredAction {
			style = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorCream).
				Background(colorIndigo)
		} else if item.accent {
			style = lipgloss.NewStyle().Foreground(colorFuchsia)
		}
		parts = append(parts, style.Render(label))
	}

	content := strings.Join(parts, strings.Repeat(" ", actionBarGap))
	return lipgloss.NewStyle().
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorMuted).
		Render(content)
}

func actionBarHitRegions(x, y int, items []actionBarItem) []hitRegion {
	if len(items) == 0 {
		return nil
	}

	barHeight := lipgloss.Height(actionBar(items, ""))
	cursor := x + 2 // border and horizontal padding
	regions := make([]hitRegion, 0, len(items))
	for index, item := range items {
		width := lipgloss.Width(item.key + "  " + item.label)
		x0 := cursor
		if index == 0 {
			x0 = x
		}
		x1 := cursor + width - 1
		if index < len(items)-1 {
			x1 += actionBarGap
		} else {
			x1 += 2 // padding and border
		}
		regions = append(regions, hitRegion{
			x0: x0, x1: x1,
			y0: y, y1: y + barHeight - 1,
			value: item.action,
		})
		cursor += width + actionBarGap
	}
	return regions
}

func usersActionBarItems() []actionBarItem {
	return []actionBarItem{
		{key: "c", label: "Create invite code", action: "create", accent: true},
		{key: "r", label: "Refresh", action: "refresh"},
		{key: "esc", label: "Back", action: "back"},
	}
}
