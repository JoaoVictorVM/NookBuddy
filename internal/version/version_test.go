package version

import "testing"

func TestVersion_StringFormatsAllFields(t *testing.T) {
	original := [3]string{Version, Commit, Date}
	t.Cleanup(func() { Version, Commit, Date = original[0], original[1], original[2] })

	Version, Commit, Date = "1.0.0", "abc1234", "2026-09-21"

	if got, want := String(), "NookBuddy v1.0.0 (abc1234, 2026-09-21)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestVersion_StringUsesDefaultsWhenUnset(t *testing.T) {
	if got, want := String(), "NookBuddy vdev (none, unknown)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
