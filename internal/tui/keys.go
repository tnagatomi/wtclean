package tui

import (
	"charm.land/bubbles/v2/table"
)

// emacsTableKeyMap returns the default bubbles/table keymap extended with
// emacs-style movement aliases (ctrl+n/p for line motion, ctrl+v / alt+v
// for page motion). Aliases are appended via Binding.SetKeys so help
// strings still describe the primary bindings.
//
// Intentionally NOT added: ctrl+a / ctrl+e (collide with g / G and the
// common tmux prefix) and ctrl+f / ctrl+b (reserved for filter-mode cursor
// movement in the upcoming filter screen).
func emacsTableKeyMap() table.KeyMap {
	km := table.DefaultKeyMap()
	km.LineDown.SetKeys(append(km.LineDown.Keys(), "ctrl+n")...)
	km.LineUp.SetKeys(append(km.LineUp.Keys(), "ctrl+p")...)
	km.PageDown.SetKeys(append(km.PageDown.Keys(), "ctrl+v")...)
	km.PageUp.SetKeys(append(km.PageUp.Keys(), "alt+v")...)
	return km
}
