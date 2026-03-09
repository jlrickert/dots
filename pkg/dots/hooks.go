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

// RunHook executes a hook script relative to the package directory.
// hookPath is the relative path from the manifest (e.g. "scripts/install.sh").
// pkgDir is the absolute path to the package directory.
func (h *HookRunner) RunHook(ctx context.Context, hookPath, pkgDir string) error {
	if hookPath == "" {
		return nil
	}

	absPath := filepath.Join(pkgDir, filepath.FromSlash(hookPath))

	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("hook script not found: %s: %w", absPath, err)
	}

	shell, args := resolveShell(absPath)

	cmd := exec.CommandContext(ctx, shell, append(args, absPath)...)
	cmd.Dir = pkgDir
	cmd.Stdout = h.Stdout
	cmd.Stderr = h.Stderr
	cmd.Env = append(os.Environ(),
		"DOTS_PACKAGE_DIR="+pkgDir,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hook %s failed: %w", hookPath, err)
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
