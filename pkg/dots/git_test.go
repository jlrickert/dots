package dots_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

// initBareRepo creates a bare Git repo in a temp directory, populates it
// with the given files, and returns the repo path. Files is a map of
// relative path to content.
func initBareRepo(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	work := filepath.Join(dir, "work")
	bare := filepath.Join(dir, "bare")

	// Create a working repo, add files, push to bare.
	run(t, "", "git", "init", work)
	run(t, work, "git", "config", "user.email", "test@test.com")
	run(t, work, "git", "config", "user.name", "Test")

	for path, content := range files {
		full := filepath.Join(work, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}

	run(t, work, "git", "add", "-A")
	run(t, work, "git", "commit", "-m", "initial")

	// Clone to bare repo.
	run(t, "", "git", "clone", "--bare", work, bare)

	return bare
}

// run executes a command and fails the test on error.
func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "command %s %v failed: %s", name, args, string(out))
}

func TestExecGitClient_CloneAndPull(t *testing.T) {
	bare := initBareRepo(t, map[string]string{
		"nvim/Dotfile.yaml": "package:\n  name: nvim\n",
		"git/Dotfile.yaml":  "package:\n  name: git\n",
	})

	ctx := context.Background()
	git := dots.NewExecGitClient()
	dest := filepath.Join(t.TempDir(), "clone")

	// Clone
	err := git.Clone(ctx, bare, dest, dots.GitCloneOpts{})
	require.NoError(t, err)

	// Verify cloned content
	_, err = os.Stat(filepath.Join(dest, "nvim", "Dotfile.yaml"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dest, "git", "Dotfile.yaml"))
	require.NoError(t, err)

	// Pull (should be a no-op, but must succeed)
	err = git.Pull(ctx, dest)
	require.NoError(t, err)
}

func TestExecGitClient_Clone_WithBranch(t *testing.T) {
	// Create a repo with a non-default branch
	dir := t.TempDir()
	work := filepath.Join(dir, "work")
	bare := filepath.Join(dir, "bare")

	run(t, "", "git", "init", work)
	run(t, work, "git", "config", "user.email", "test@test.com")
	run(t, work, "git", "config", "user.name", "Test")
	run(t, work, "git", "checkout", "-b", "develop")

	require.NoError(t, os.MkdirAll(filepath.Join(work, "zsh"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(work, "zsh", "Dotfile.yaml"),
		[]byte("package:\n  name: zsh\n"), 0o644,
	))

	run(t, work, "git", "add", "-A")
	run(t, work, "git", "commit", "-m", "initial on develop")
	run(t, "", "git", "clone", "--bare", work, bare)

	ctx := context.Background()
	git := dots.NewExecGitClient()
	dest := filepath.Join(t.TempDir(), "clone")

	err := git.Clone(ctx, bare, dest, dots.GitCloneOpts{Branch: "develop"})
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dest, "zsh", "Dotfile.yaml"))
	require.NoError(t, err)
}

func TestExecGitClient_Clone_InvalidURL(t *testing.T) {
	ctx := context.Background()
	git := dots.NewExecGitClient()
	dest := filepath.Join(t.TempDir(), "clone")

	err := git.Clone(ctx, "/nonexistent/repo", dest, dots.GitCloneOpts{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "git clone")
}

func TestFsRepo_AddTap_ClonesRepo(t *testing.T) {
	bare := initBareRepo(t, map[string]string{
		"nvim/Dotfile.yaml": "package:\n  name: nvim\n",
		"git/Dotfile.yaml":  "package:\n  name: git\n",
	})

	dir := t.TempDir()
	repo := dots.NewFsRepo(
		filepath.Join(dir, "config"),
		filepath.Join(dir, "state"),
		nil, // real ExecGitClient
	)

	ctx := context.Background()
	err := repo.AddTap(ctx, dots.TapInfo{
		Name: "personal",
		URL:  bare,
	})
	require.NoError(t, err)

	// Verify packages are discoverable
	pkgs, err := repo.ListPackages(ctx, "personal")
	require.NoError(t, err)
	require.Len(t, pkgs, 2)
	require.Equal(t, "git", pkgs[0].Name)
	require.Equal(t, "nvim", pkgs[1].Name)

	// Verify manifest is readable
	data, err := repo.ReadManifest(ctx, "personal", "nvim")
	require.NoError(t, err)
	require.Contains(t, string(data), "name: nvim")
}

func TestFsRepo_AddTap_LocalPath(t *testing.T) {
	bare := initBareRepo(t, map[string]string{
		"tmux/Dotfile.yaml": "package:\n  name: tmux\n",
	})

	dir := t.TempDir()
	repo := dots.NewFsRepo(
		filepath.Join(dir, "config"),
		filepath.Join(dir, "state"),
		nil,
	)

	ctx := context.Background()
	// Use file:// scheme
	err := repo.AddTap(ctx, dots.TapInfo{
		Name: "local",
		URL:  "file://" + bare,
	})
	require.NoError(t, err)

	pkgs, err := repo.ListPackages(ctx, "local")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	require.Equal(t, "tmux", pkgs[0].Name)
}

func TestFsRepo_UpdateTap_Pulls(t *testing.T) {
	// Set up bare repo and clone via FsRepo
	dir := t.TempDir()
	work := filepath.Join(dir, "work")
	bare := filepath.Join(dir, "bare")

	run(t, "", "git", "init", work)
	run(t, work, "git", "config", "user.email", "test@test.com")
	run(t, work, "git", "config", "user.name", "Test")

	require.NoError(t, os.MkdirAll(filepath.Join(work, "zsh"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(work, "zsh", "Dotfile.yaml"),
		[]byte("package:\n  name: zsh\n"), 0o644,
	))
	run(t, work, "git", "add", "-A")
	run(t, work, "git", "commit", "-m", "initial")
	run(t, "", "git", "clone", "--bare", work, bare)

	repoDir := t.TempDir()
	repo := dots.NewFsRepo(
		filepath.Join(repoDir, "config"),
		filepath.Join(repoDir, "state"),
		nil,
	)

	ctx := context.Background()
	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: bare})
	require.NoError(t, err)

	// Add a new package to the source, push to bare
	require.NoError(t, os.MkdirAll(filepath.Join(work, "vim"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(work, "vim", "Dotfile.yaml"),
		[]byte("package:\n  name: vim\n"), 0o644,
	))
	run(t, work, "git", "add", "-A")
	run(t, work, "git", "commit", "-m", "add vim")
	run(t, work, "git", "push", bare)

	// Update should pull the new content
	err = repo.UpdateTap(ctx, "personal")
	require.NoError(t, err)

	// Now vim should be discoverable
	pkgs, err := repo.ListPackages(ctx, "personal")
	require.NoError(t, err)

	names := make([]string, len(pkgs))
	for i, p := range pkgs {
		names[i] = p.Name
	}
	require.Contains(t, names, "vim")
	require.Contains(t, names, "zsh")
}
