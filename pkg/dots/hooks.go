package dots

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// HookRunner executes lifecycle hook scripts from package manifests.
type HookRunner struct {
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
		shell, args := resolveShell(absPath)
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
