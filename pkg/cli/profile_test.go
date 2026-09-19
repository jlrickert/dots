package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProfileApply_DryRun(t *testing.T) {
	sb := NewSandbox(t)
	_, stderr, code := RunCommand(t, sb, "profile", "create", "empty")
	require.Equal(t, 0, code, stderr)

	stdout, stderr, code := RunCommand(t, sb, "profile", "apply", "empty", "--dry-run")
	require.Equal(t, 0, code, stderr)
	require.Contains(t, stdout, "Dry run — no changes made:")
	require.NotContains(t, stdout, "Applied profile")
}

func TestProfileApply_MissingProfile(t *testing.T) {
	sb := NewSandbox(t)
	_, stderr, code := RunCommand(t, sb, "profile", "apply", "nope", "--dry-run")
	require.NotEqual(t, 0, code)
	require.Contains(t, stderr, `profile "nope"`)
}

func TestProfileSwitch_Removed(t *testing.T) {
	sb := NewSandbox(t)
	stdout, _, _ := RunCommand(t, sb, "profile", "--help")
	require.NotContains(t, stdout, "switch")
	require.Contains(t, stdout, "apply")
}
