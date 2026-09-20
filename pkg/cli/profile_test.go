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

func TestProfileShow_ResolvedWithSources(t *testing.T) {
	sb := NewSandbox(t)
	for _, args := range [][]string{
		{"profile", "create", "base"},
		{"profile", "add", "base", "personal/git", "personal/tmux"},
		{"profile", "create", "work", "--extends", "base"},
		{"profile", "add", "work", "personal/nvim"},
		{"profile", "exclude", "work", "personal/tmux", "personal/none"},
	} {
		_, stderr, code := RunCommand(t, sb, args...)
		require.Equal(t, 0, code, "%v: %s", args, stderr)
	}

	stdout, stderr, code := RunCommand(t, sb, "profile", "show", "work")
	require.Equal(t, 0, code, stderr)
	require.Contains(t, stdout, "Extends:  base\n")
	require.Contains(t, stdout, "Exclude:  personal/tmux, personal/none\n")
	require.Contains(t, stdout, "Resolved:\n  personal/git   (base)\n  personal/nvim  (work)\n")
	require.NotContains(t, stdout, "personal/tmux  (")
	require.Contains(t, stdout, "Unmatched excludes (no parent provides these):\n  personal/none\n")
}

func TestProfileExtend_CycleFails(t *testing.T) {
	sb := NewSandbox(t)
	for _, args := range [][]string{
		{"profile", "create", "base"},
		{"profile", "create", "work", "--extends", "base"},
	} {
		_, stderr, code := RunCommand(t, sb, args...)
		require.Equal(t, 0, code, "%v: %s", args, stderr)
	}

	_, stderr, code := RunCommand(t, sb, "profile", "extend", "base", "work")
	require.NotEqual(t, 0, code)
	require.Contains(t, stderr, "base → work → base")
}
