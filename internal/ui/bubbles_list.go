package ui

import (
	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
)

type bluffListItem struct {
	title       string
	description string
}

func (item bluffListItem) Title() string       { return item.title }
func (item bluffListItem) Description() string { return item.description }
func (item bluffListItem) FilterValue() string { return item.title }

// bluffListView uses Bubbles' default two-line list delegate. Bluff keeps its
// global help bar, so the component only owns item layout and pagination.
func bluffListView(title string, items []list.Item, selected, width, height int) string {
	return bluffListViewWithTitleStyle(title, items, selected, width, height, nil)
}

func bluffListViewWithTitleStyle(title string, items []list.Item, selected, width, height int, titleStyle *lipgloss.Style) string {
	delegate := list.NewDefaultDelegate()
	model := list.New(items, delegate, max(width, 1), max(height, 4))
	model.Title = title
	if titleStyle != nil {
		model.Styles.Title = *titleStyle
	}
	model.SetShowStatusBar(false)
	model.SetFilteringEnabled(false)
	model.SetShowHelp(false)
	if len(items) > 0 {
		model.Select(max(0, min(selected, len(items)-1)))
	}
	return model.View()
}
