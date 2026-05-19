package tui

import (
	"slices"

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

// worktreeTableKeyMap is the emacs keymap with two defaults stripped so
// they don't collide with the worktree-screen actions:
//   - "space" is removed from PageDown so it can toggle selection.
//   - "d" is removed from HalfPageDown so it can open the delete
//     confirmation screen.
//
// The repo table keeps the unmodified defaults since it has no selection
// or delete concept.
func worktreeTableKeyMap() table.KeyMap {
	km := emacsTableKeyMap()
	km.PageDown.SetKeys(without(km.PageDown.Keys(), "space")...)
	km.HalfPageDown.SetKeys(without(km.HalfPageDown.Keys(), "d")...)
	return km
}

func without(keys []string, drop string) []string {
	return slices.DeleteFunc(slices.Clone(keys), func(k string) bool { return k == drop })
}
