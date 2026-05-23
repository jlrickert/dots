package dots_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

// writeScript drops a hook script into a temp package dir and returns the
// package dir and the relative hook value. The script is not chmod +x —
// dots reads it itself rather than asking the kernel to exec it.
func writeScript(t *testing.T, body string) (pkgDir, hookValue string) {
	t.Helper()
	pkgDir = t.TempDir()
	rel := filepath.Join("scripts", "hook.sh")
	full := filepath.Join(pkgDir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	return pkgDir, rel
}

func newRunner(t *testing.T) (*dots.HookRunner, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	rt, err := toolkit.NewOsRuntime()
	require.NoError(t, err)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return &dots.HookRunner{Runtime: rt, Stdout: stdout, Stderr: stderr}, stdout, stderr
}

func TestRunHook_EmptyValueIsNoop(t *testing.T) {
	runner, stdout, stderr := newRunner(t)
	require.NoError(t, runner.RunHook(context.Background(), "", t.TempDir()))
	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
}

func TestRunHook_ShebangShRunsUnderSh(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shebang behavior is Unix-only")
	}
	if _, err := exec.LookPath("/bin/sh"); err != nil {
		t.Skipf("/bin/sh not available: %v", err)
	}
	// $SHELL points at zsh to prove the shebang wins over the env fallback.
	t.Setenv("SHELL", "/bin/zsh")

	pkgDir, hook := writeScript(t, "#!/bin/sh\necho hello-from-sh\n")
	runner, stdout, _ := newRunner(t)
	require.NoError(t, runner.RunHook(context.Background(), hook, pkgDir))
	require.Contains(t, stdout.String(), "hello-from-sh")
}

func TestRunHook_ShebangBashHonoredOverZsh(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash shebang behavior is Unix-only")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not on PATH: %v", err)
	}
	if _, err := exec.LookPath("/usr/bin/env"); err != nil {
		t.Skipf("/usr/bin/env not available: %v", err)
	}
	// Repro the issue's failure mode: SHELL=/bin/zsh would parse-error on
	// `shopt`. If the shebang is honored the script runs under bash and prints.
	t.Setenv("SHELL", "/bin/zsh")

	body := "#!/usr/bin/env bash\nshopt -s nullglob\necho bash-ok\n"
	pkgDir, hook := writeScript(t, body)
	runner, stdout, stderr := newRunner(t)
	err := runner.RunHook(context.Background(), hook, pkgDir)
	require.NoError(t, err, "stderr=%q", stderr.String())
	require.Contains(t, stdout.String(), "bash-ok")
}

func TestRunHook_NoShebangFallsBackToShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fallback test targets Unix")
	}
	if _, err := exec.LookPath("/bin/sh"); err != nil {
		t.Skipf("/bin/sh not available: %v", err)
	}
	t.Setenv("SHELL", "/bin/sh")

	pkgDir, hook := writeScript(t, "echo no-shebang\n")
	runner, stdout, _ := newRunner(t)
	require.NoError(t, runner.RunHook(context.Background(), hook, pkgDir))
	require.Contains(t, stdout.String(), "no-shebang")
}

func TestRunHook_InlineCommandUnchanged(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("inline branch uses /bin/sh on Unix; Windows uses cmd.exe and a different syntax")
	}
	if _, err := exec.LookPath("/bin/sh"); err != nil {
		t.Skipf("/bin/sh not available: %v", err)
	}
	t.Setenv("SHELL", "/bin/sh")

	runner, stdout, _ := newRunner(t)
	require.NoError(t, runner.RunHook(context.Background(), "echo inline-value", t.TempDir()))
	require.Contains(t, stdout.String(), "inline-value")
}

func TestRunHook_NonZeroExitWrapsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX exit semantics")
	}
	pkgDir, hook := writeScript(t, "#!/bin/sh\nexit 3\n")
	runner, _, _ := newRunner(t)
	err := runner.RunHook(context.Background(), hook, pkgDir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "hook "+hook+" failed:")
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr), "expected wrapped *exec.ExitError, got %T", err)
}

func TestRunHook_HookValueIsTrimmed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-only")
	}
	if _, err := exec.LookPath("/bin/sh"); err != nil {
		t.Skipf("/bin/sh not available: %v", err)
	}
	pkgDir, hook := writeScript(t, "#!/bin/sh\necho trimmed\n")
	runner, stdout, _ := newRunner(t)
	padded := "  " + hook + "  "
	require.NoError(t, runner.RunHook(context.Background(), padded, pkgDir))
	require.True(t, strings.Contains(stdout.String(), "trimmed"))
}
