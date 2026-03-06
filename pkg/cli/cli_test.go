package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	sb := NewSandbox(t)
	stdout, _, code := RunCommand(t, sb, "--version")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "dots version dev")
}

func TestHelp(t *testing.T) {
	sb := NewSandbox(t)
	stdout, _, code := RunCommand(t, sb, "--help")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "brew-style dotfile package manager")
}

func TestCompletionZsh(t *testing.T) {
	sb := NewSandbox(t)
	stdout, _, code := RunCommand(t, sb, "completion", "zsh")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "compdef")
}

func TestCompletionBash(t *testing.T) {
	sb := NewSandbox(t)
	stdout, _, code := RunCommand(t, sb, "completion", "bash")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "bash completion")
}

func TestGlobalFlags(t *testing.T) {
	sb := NewSandbox(t)
	stdout, _, code := RunCommand(t, sb, "--help")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "brew-style dotfile package manager")
}
