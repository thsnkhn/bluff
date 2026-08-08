package ui

import "charm.land/huh/v2"

// newHuhForm keeps all application forms on Huh's standard form model. The
// footer belongs to Bluff, so Huh's per-field help stays hidden while its
// built-in validation remains visible below the active field.
//
// TODO: Keep accessibility options here too if Bluff adds a user preference
// for Huh's screen-reader mode.
func newHuhForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).WithShowHelp(false)
}

// newHuhInput is the shared stock Huh input used by all ordinary text fields.
// Composite controls such as chip grids may still wrap Huh inputs when they
// need a layout that a single Input cannot provide.
func newHuhInput(
	title string,
	description string,
	placeholder string,
	value *string,
	charLimit int,
	password bool,
	validate func(string) error,
) *huh.Input {
	input := huh.NewInput().
		Title(title).
		Description(description).
		Placeholder(placeholder).
		Value(value)
	if validate != nil {
		input.Validate(validate)
	}
	if charLimit > 0 {
		input.CharLimit(charLimit)
	}
	if password {
		input.EchoMode(huh.EchoModePassword)
	}
	return input
}
