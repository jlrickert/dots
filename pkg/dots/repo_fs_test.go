package dots_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

func newFsRepo(t *testing.T) *dots.FsRepo {
	t.Helper()
	dir := t.TempDir()
	return dots.NewFsRepo(
		filepath.Join(dir, "config"),
		filepath.Join(dir, "state"),
	)
}

func TestFsRepo_Name(t *testing.T) {
	repo := newFsRepo(t)
	require.Equal(t, "fs", repo.Name())
}

// --- Tap management ---

func TestFsRepo_AddAndGetTap(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	err := repo.AddTap(ctx, dots.TapInfo{
		Name: "personal",
		URL:  "git@github.com:user/dotfiles.git",
	})
	require.NoError(t, err)

	tap, err := repo.GetTap(ctx, "personal")
	require.NoError(t, err)
	require.Equal(t, "personal", tap.Name)
	require.Equal(t, "git@github.com:user/dotfiles.git", tap.URL)
}

func TestFsRepo_AddTap_Duplicate(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	err := repo.AddTap(ctx, dots.TapInfo{Name: "dup", URL: "url"})
	require.NoError(t, err)

	err = repo.AddTap(ctx, dots.TapInfo{Name: "dup", URL: "url2"})
	require.ErrorIs(t, err, dots.ErrExist)
}

func TestFsRepo_GetTap_NotFound(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	_, err := repo.GetTap(ctx, "missing")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

func TestFsRepo_ListTaps(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	// Empty list
	taps, err := repo.ListTaps(ctx)
	require.NoError(t, err)
	require.Empty(t, taps)

	err = repo.AddTap(ctx, dots.TapInfo{Name: "beta", URL: "b"})
	require.NoError(t, err)
	err = repo.AddTap(ctx, dots.TapInfo{Name: "alpha", URL: "a"})
	require.NoError(t, err)

	taps, err = repo.ListTaps(ctx)
	require.NoError(t, err)
	require.Len(t, taps, 2)
	require.Equal(t, "alpha", taps[0].Name)
	require.Equal(t, "beta", taps[1].Name)
}

func TestFsRepo_RemoveTap(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	err := repo.AddTap(ctx, dots.TapInfo{Name: "removeme", URL: "url"})
	require.NoError(t, err)

	err = repo.RemoveTap(ctx, "removeme")
	require.NoError(t, err)

	_, err = repo.GetTap(ctx, "removeme")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

func TestFsRepo_RemoveTap_NotFound(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	err := repo.RemoveTap(ctx, "missing")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

func TestFsRepo_UpdateTap(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)

	// Should be a no-op but not error
	err = repo.UpdateTap(ctx, "personal")
	require.NoError(t, err)
}

func TestFsRepo_UpdateTap_NotFound(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	err := repo.UpdateTap(ctx, "missing")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

// --- Package discovery ---

func TestFsRepo_ListPackages(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)

	// Create package directories with Dotfile.yaml
	tapDir := filepath.Join(repo.StateDir, "taps", "personal")
	nvimDir := filepath.Join(tapDir, "nvim")
	gitDir := filepath.Join(tapDir, "git")
	require.NoError(t, os.MkdirAll(nvimDir, 0o755))
	require.NoError(t, os.MkdirAll(gitDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "Dotfile.yaml"),
		[]byte("package:\n  name: nvim\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "Dotfile.yaml"),
		[]byte("package:\n  name: git\n"), 0o644))

	packages, err := repo.ListPackages(ctx, "personal")
	require.NoError(t, err)
	require.Len(t, packages, 2)
	require.Equal(t, "git", packages[0].Name)
	require.Equal(t, "nvim", packages[1].Name)
}

func TestFsRepo_ListPackages_TapNotFound(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	_, err := repo.ListPackages(ctx, "missing")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

func TestFsRepo_ReadManifest(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)

	tapDir := filepath.Join(repo.StateDir, "taps", "personal")
	nvimDir := filepath.Join(tapDir, "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0o755))

	manifest := []byte("package:\n  name: nvim\nlinks:\n  init.lua: \"@config/nvim/init.lua\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "Dotfile.yaml"), manifest, 0o644))

	data, err := repo.ReadManifest(ctx, "personal", "nvim")
	require.NoError(t, err)
	require.Equal(t, manifest, data)
}

func TestFsRepo_ReadManifest_NotFound(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)

	_, err = repo.ReadManifest(ctx, "personal", "missing")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

// --- Lockfile ---

func TestFsRepo_Lockfile_ReadWrite(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	// Initially not exist
	_, err := repo.ReadLockfile(ctx)
	require.ErrorIs(t, err, dots.ErrNotExist)

	lf := &dots.Lockfile{
		State: dots.LockfileState{
			LinkStrategy: dots.LinkSymlink,
			Platform:     "darwin-arm64",
		},
		Installed: []dots.InstalledPackage{
			{Package: "personal/nvim", Tap: "personal"},
		},
	}

	err = repo.WriteLockfile(ctx, lf)
	require.NoError(t, err)

	got, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	require.Equal(t, "darwin-arm64", got.State.Platform)
	require.Len(t, got.Installed, 1)
	require.Equal(t, "personal/nvim", got.Installed[0].Package)
}

// --- Backups ---

func TestFsRepo_Backup_And_Restore(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	data := []byte("original file content")
	err := repo.BackupFile(ctx, "home/.gitconfig", data)
	require.NoError(t, err)

	restored, err := repo.RestoreFile(ctx, "home/.gitconfig")
	require.NoError(t, err)
	require.Equal(t, data, restored)
}

func TestFsRepo_RestoreFile_NotFound(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	_, err := repo.RestoreFile(ctx, "missing")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

func TestFsRepo_ListBackups(t *testing.T) {
	repo := newFsRepo(t)
	ctx := context.Background()

	// Empty initially
	paths, err := repo.ListBackups(ctx)
	require.NoError(t, err)
	require.Empty(t, paths)

	require.NoError(t, repo.BackupFile(ctx, "a/file1", []byte("1")))
	require.NoError(t, repo.BackupFile(ctx, "b/file2", []byte("2")))

	paths, err = repo.ListBackups(ctx)
	require.NoError(t, err)
	require.Len(t, paths, 2)
	require.Equal(t, "a/file1", paths[0])
	require.Equal(t, "b/file2", paths[1])
}
