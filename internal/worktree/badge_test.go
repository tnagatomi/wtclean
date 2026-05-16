package worktree

import "testing"

func TestBadgeString(t *testing.T) {
	cases := []struct {
		b    Badge
		want string
	}{
		{BadgePrimary, "primary"},
		{BadgeMerged, "merged"},
		{BadgeGone, "gone"},
		{BadgeDirty, "dirty"},
		{BadgeUnpushed, "unpushed"},
		{BadgeLocked, "locked"},
		{BadgeMissing, "missing"},
		{Badge(-1), ""},
	}
	for _, c := range cases {
		if got := c.b.String(); got != c.want {
			t.Errorf("Badge(%d).String() = %q, want %q", c.b, got, c.want)
		}
	}
}
