package ui

import "strings"

func (m Model) isSearchableScreen() bool {
	switch m.screen {
	case usersScreen, tablesScreen, tableDetailScreen, formatsScreen, formatDetailScreen, playersScreen, gamesScreen, gameDetailScreen:
		return true
	default:
		return false
	}
}

// searchActionBar keeps search in the same intrinsic action bar used by list screens.
func searchActionBar(items []actionBarItem, hovered string, active bool, query string) string {
	if !active {
		return actionBar(items, hovered)
	}
	copyItems := append([]actionBarItem(nil), items...)
	for index := range copyItems {
		if copyItems[index].action == "search" {
			label := "Search"
			if query != "" {
				label += ": " + query
			}
			copyItems[index].label = label + "▌"
			copyItems[index].accent = true
		}
	}
	return actionBar(copyItems, "search")
}

func searchMatches(query, value string) bool {
	query = strings.TrimSpace(query)
	return query == "" || strings.Contains(strings.ToLower(value), strings.ToLower(query))
}

func trimLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

func (m Model) visibleUserIndices() []int {
	indices := make([]int, 0, len(m.users))
	for index, user := range m.users {
		if searchMatches(m.searchQuery, user.Username+" "+user.Role) {
			indices = append(indices, index)
		}
	}
	return indices
}

func (m Model) visibleTableIndices() []int {
	indices := make([]int, 0, len(m.tables))
	for index, table := range m.tables {
		if searchMatches(m.searchQuery, table.Name+" "+table.HostUsername) {
			indices = append(indices, index)
		}
	}
	return indices
}

func (m Model) visibleFormatIndices() []int {
	indices := make([]int, 0, len(m.table.Formats))
	for index, format := range m.table.Formats {
		if searchMatches(m.searchQuery, format.Name) {
			indices = append(indices, index)
		}
	}
	return indices
}

func (m Model) visiblePlayerIndices() []int {
	indices := make([]int, 0, len(m.table.Players))
	for index, player := range m.table.Players {
		if searchMatches(m.searchQuery, player.Name) {
			indices = append(indices, index)
		}
	}
	return indices
}

func (m Model) visibleGameIndices() []int {
	indices := make([]int, 0, len(m.table.Games))
	for index, game := range m.table.Games {
		if searchMatches(m.searchQuery, game.Format.Name+" "+game.Date+" "+game.Status) {
			indices = append(indices, index)
		}
	}
	return indices
}

func moveVisible(current *int, indices []int, direction int) {
	if len(indices) == 0 {
		*current = 0
		return
	}
	position := 0
	for index, value := range indices {
		if value == *current {
			position = index
			break
		}
	}
	position = max(0, min(position+direction, len(indices)-1))
	*current = indices[position]
}
