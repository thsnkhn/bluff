package ui

import (
	"strconv"
	"strings"

	"github.com/thsnkhn/bluff/internal/api"
)

// newerRelease only compares stable, three-part client versions. Development
// builds and malformed server values intentionally skip automatic updates.
func (m Model) newerRelease(latest string) *api.ClientRelease {
	if !newerClientVersion(m.build.Version, latest) {
		return nil
	}
	return &api.ClientRelease{Version: normalizeClientVersion(latest)}
}

func newerClientVersion(current, latest string) bool {
	currentParts, ok := parseClientVersion(current)
	if !ok {
		return false
	}
	latestParts, ok := parseClientVersion(latest)
	if !ok {
		return false
	}
	for index := range currentParts {
		if latestParts[index] != currentParts[index] {
			return latestParts[index] > currentParts[index]
		}
	}
	return false
}

func normalizeClientVersion(version string) string {
	return "v" + strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func parseClientVersion(version string) ([3]int, bool) {
	var parsed [3]int
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	parts := strings.Split(version, ".")
	if len(parts) != len(parsed) {
		return parsed, false
	}
	for index, part := range parts {
		if part == "" {
			return parsed, false
		}
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return parsed, false
		}
		parsed[index] = value
	}
	return parsed, true
}
