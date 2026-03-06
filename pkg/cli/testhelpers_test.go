package cli_test

import (
	"context"
	"embed"
	"testing"

	tu "github.com/jlrickert/cli-toolkit/sandbox"
	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/dots/pkg/cli"
)

//go:embed all:data/**
var testdata embed.FS

// NewSandbox creates a test sandbox with embedded fixture data.
func NewSandbox(t *testing.T, opts ...tu.Option) *tu.Sandbox {
	return tu.NewSandbox(t, &tu.Options{
		Data: testdata,
		Home: "/home/testuser",
		User: "testuser",
	}, opts...)
}

// NewProcess creates a Process that runs the dots CLI with the given args.
func NewProcess(t *testing.T, isTTY bool, args ...string) *tu.Process {
	return tu.NewProcess(func(ctx context.Context, rt *toolkit.Runtime) (int, error) {
		return cli.Run(ctx, rt, args)
	}, isTTY)
}

// RunCommand is a convenience that runs a dots CLI command in a sandbox and
// returns stdout, stderr, and exit code.
func RunCommand(t *testing.T, sb *tu.Sandbox, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	proc := NewProcess(t, false, args...)
	outBuf := proc.CaptureStdout()
	errBuf := proc.CaptureStderr()
	result := proc.Run(sb.Context(), sb.Runtime())
	return outBuf.String(), errBuf.String(), result.ExitCode
}
