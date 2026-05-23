package dots

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jlrickert/cli-toolkit/toolkit"
)

// HookRunner executes lifecycle hook scripts from package manifests.
type HookRunner struct {
	// Runtime is the I/O seam used to read script shebangs. When nil, hook
	// dispatch still works but bypasses the test runtime's jailed FS.
	Runtime *toolkit.Runtime
	// Stdout and Stderr receive hook output.
	Stdout io.Writer
	Stderr io.Writer
}

// RunHook executes a hook, which may be either a script file path relative to
// the package directory (e.g. "scripts/install.sh") or an inline shell command
// (e.g. "launchctl load ~/Library/LaunchAgents/foo.plist").
func (h *HookRunner) RunHook(ctx context.Context, hookValue, pkgDir string) error {
	if hookValue == "" {
		return nil
	}

	env := append(os.Environ(), "DOTS_PACKAGE_DIR="+pkgDir)

	// Check if it's a file path by joining with pkgDir and stat-ing.
	absPath := filepath.Join(pkgDir, filepath.FromSlash(strings.TrimSpace(hookValue)))
	if info, err := os.Stat(absPath); err == nil && !info.IsDir() {
		shell, args := h.resolveInterpreter(absPath)
		cmd := exec.CommandContext(ctx, shell, append(args, absPath)...)
		cmd.Dir = pkgDir
		cmd.Stdout = h.Stdout
		cmd.Stderr = h.Stderr
		cmd.Env = env
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("hook %s failed: %w", hookValue, err)
		}
		return nil
	}

	// Otherwise treat as an inline shell command.
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	if runtime.GOOS == "windows" {
		shell = "cmd.exe"
	}

	cmd := exec.CommandContext(ctx, shell, "-c", hookValue)
	cmd.Dir = pkgDir
	cmd.Stdout = h.Stdout
	cmd.Stderr = h.Stderr
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hook command failed: %w", err)
	}
	return nil
}

// resolveInterpreter picks the program to run a hook script with. On Unix it
// honors the script's `#!` line so a hook tagged `#!/usr/bin/env bash` runs
// under bash regardless of the user's $SHELL. The kernel's execve shebang
// path is skipped because the script is being passed as an argument rather
// than exec'd directly, so we reproduce that lookup here. Any failure to
// read or parse the shebang falls back to resolveShell, which preserves the
// previous $SHELL-driven behavior. On Windows shebangs are ignored entirely
// — extension-based dispatch in resolveShell takes precedence.
func (h *HookRunner) resolveInterpreter(scriptPath string) (string, []string) {
	if runtime.GOOS == "windows" {
		return resolveShell(scriptPath)
	}
	data, err := h.readScript(scriptPath)
	if err != nil {
		return resolveShell(scriptPath)
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		data = data[:i]
	}
	line := strings.TrimRight(string(data), "\r")
	if !strings.HasPrefix(line, "#!") {
		return resolveShell(scriptPath)
	}
	fields := strings.Fields(strings.TrimPrefix(line, "#!"))
	if len(fields) == 0 {
		return resolveShell(scriptPath)
	}
	return fields[0], fields[1:]
}

// readScript reads scriptPath through the toolkit Runtime so the test
// runtime's jailed FS sees the access. When no Runtime is wired the read
// falls back to the host FS — the same posture HookRunner already has for
// os.Stat and exec.
func (h *HookRunner) readScript(scriptPath string) ([]byte, error) {
	if h != nil && h.Runtime != nil {
		return h.Runtime.ReadFile(scriptPath)
	}
	return os.ReadFile(scriptPath)
}

// resolveShell determines the shell and arguments to use for a hook script.
func resolveShell(scriptPath string) (string, []string) {
	ext := strings.ToLower(filepath.Ext(scriptPath))

	if runtime.GOOS == "windows" {
		if ext == ".ps1" {
			return "powershell.exe", []string{"-ExecutionPolicy", "Bypass", "-File"}
		}
		if ext == ".cmd" || ext == ".bat" {
			return "cmd.exe", []string{"/C"}
		}
	}

	// Unix: use $SHELL or /bin/sh
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return shell, nil
}
