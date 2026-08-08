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

// popupActionBar is the compact, borderless action strip used in popup headers.
func popupActionBar(items []actionBarItem, hoveredAction string) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		label := item.key + "  " + item.label
		style := mutedStyle
		if item.action == hoveredAction {
			style = lipgloss.NewStyle().Bold(true).Foreground(colorCream).Background(colorIndigo)
		} else if item.accent {
			style = lipgloss.NewStyle().Foreground(colorFuchsia)
		}
		parts = append(parts, style.Render(label))
	}
	return strings.Join(parts, strings.Repeat(" ", 3))
}

func popupHeader(title string, width int) string {
	return popupHeaderWithRight(title, "", width)
}

// popupHeaderWithRight keeps the title on the left and an optional value on
// the far right, with the same slash rule used by the rest of Bluff's chrome.
func popupHeaderWithRight(title, right string, width int) string {
	titleView := brandStyle.Render(title)
	rightWidth := lipgloss.Width(right)
	used := lipgloss.Width(titleView) + 1
	if rightWidth > 0 {
		used += rightWidth + 1
	}
	ruleWidth := max(width-used, 3)
	rule := lipgloss.NewStyle().Foreground(colorIndigo).Render(strings.Repeat("/", ruleWidth))
	if rightWidth == 0 {
		return titleView + " " + rule
	}
	return titleView + " " + rule + " " + right
}

// popupActionFooter keeps popup actions discoverable without competing with
// the title. It uses the same compact, borderless treatment as the app help bar.
func popupActionFooter(items []actionBarItem, width int) string {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Render(popupActionBar(items, ""))
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
		{key: "/", label: "Search", action: "search"},
		{key: "esc", label: "Back", action: "back"},
	}
}
