package cli_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/dots/pkg/cli"
	"github.com/stretchr/testify/require"
)

func runDots(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	stream := &toolkit.Stream{
		In:  nil,
		Out: &outBuf,
		Err: &errBuf,
	}
	dir := t.TempDir()
	rt, err := toolkit.NewTestRuntime(dir, dir, "testuser",
		toolkit.WithRuntimeStream(stream),
	)
	require.NoError(t, err)

	code, _ := cli.Run(context.Background(), rt, args)
	return outBuf.String(), errBuf.String(), code
}

func TestVersion(t *testing.T) {
	stdout, _, code := runDots(t, "--version")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "dots version dev")
}

func TestHelp(t *testing.T) {
	stdout, _, code := runDots(t, "--help")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "brew-style dotfile package manager")
}

func TestCompletionZsh(t *testing.T) {
	stdout, _, code := runDots(t, "completion", "zsh")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "compdef")
}
