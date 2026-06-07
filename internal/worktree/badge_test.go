package worktree

import "testing"

func TestBadgeString(t *testing.T) {
	cases := []struct {
		b    Badge
		want string
	}{
		{BadgePrimary, "primary"},
		{BadgeMerged, "merged"},
		{BadgeUpstreamGone, "upstream-gone"},
		{BadgeUncommitted, "uncommitted"},
		{BadgeUnpushed, "unpushed"},
		{BadgeLocked, "locked"},
		{BadgeNoDir, "no-dir"},
		{Badge(-1), ""},
	}
	for _, c := range cases {
		if got := c.b.String(); got != c.want {
			t.Errorf("Badge(%d).String() = %q, want %q", c.b, got, c.want)
		}
	}
}
