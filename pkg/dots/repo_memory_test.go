package dots_test

import (
	"context"
	"testing"
	"time"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

func newTestRepo(t *testing.T) *dots.MemoryRepo {
	t.Helper()
	return dots.NewMemoryRepo()
}

func TestMemoryRepo_Name(t *testing.T) {
	repo := newTestRepo(t)
	require.Equal(t, "memory", repo.Name())
}

// --- Tap management ---

func TestMemoryRepo_AddAndListTaps(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "git@github.com:me/dotfiles.git"})
	require.NoError(t, err)
	err = repo.AddTap(ctx, dots.TapInfo{Name: "work", URL: "git@github.com:corp/dotfiles.git"})
	require.NoError(t, err)

	taps, err := repo.ListTaps(ctx)
	require.NoError(t, err)
	require.Len(t, taps, 2)
	require.Equal(t, "personal", taps[0].Name)
	require.Equal(t, "work", taps[1].Name)
}

func TestMemoryRepo_AddTap_Duplicate(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)

	err = repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url2"})
	require.ErrorIs(t, err, dots.ErrExist)
}

func TestMemoryRepo_GetTap(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "git@github.com:me/dotfiles.git"})
	require.NoError(t, err)

	tap, err := repo.GetTap(ctx, "personal")
	require.NoError(t, err)
	require.Equal(t, "personal", tap.Name)
	require.Equal(t, "git@github.com:me/dotfiles.git", tap.URL)
}

func TestMemoryRepo_GetTap_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	_, err := repo.GetTap(ctx, "missing")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

func TestMemoryRepo_RemoveTap(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)

	err = repo.RemoveTap(ctx, "personal")
	require.NoError(t, err)

	taps, err := repo.ListTaps(ctx)
	require.NoError(t, err)
	require.Empty(t, taps)
}

func TestMemoryRepo_RemoveTap_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	err := repo.RemoveTap(ctx, "missing")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

func TestMemoryRepo_UpdateTap(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)

	err = repo.UpdateTap(ctx, "personal")
	require.NoError(t, err)
}

func TestMemoryRepo_UpdateTap_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	err := repo.UpdateTap(ctx, "missing")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

// --- Package discovery ---

func TestMemoryRepo_ListPackages(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)

	err = repo.AddPackage("personal", dots.PackageInfo{
		Tap: "personal", Name: "nvim", Dir: "nvim",
	}, []byte("package:\n  name: nvim\n"))
	require.NoError(t, err)

	err = repo.AddPackage("personal", dots.PackageInfo{
		Tap: "personal", Name: "git", Dir: "git",
	}, []byte("package:\n  name: git\n"))
	require.NoError(t, err)

	pkgs, err := repo.ListPackages(ctx, "personal")
	require.NoError(t, err)
	require.Len(t, pkgs, 2)
	require.Equal(t, "nvim", pkgs[0].Name)
	require.Equal(t, "git", pkgs[1].Name)
}

func TestMemoryRepo_ListPackages_TapNotFound(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	_, err := repo.ListPackages(ctx, "missing")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

func TestMemoryRepo_ReadManifest(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)

	manifest := []byte("package:\n  name: nvim\n  version: 1.0.0\n")
	err = repo.AddPackage("personal", dots.PackageInfo{
		Tap: "personal", Name: "nvim", Dir: "nvim",
	}, manifest)
	require.NoError(t, err)

	data, err := repo.ReadManifest(ctx, "personal", "nvim")
	require.NoError(t, err)
	require.Equal(t, manifest, data)
}

func TestMemoryRepo_ReadManifest_PackageNotFound(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)

	_, err = repo.ReadManifest(ctx, "personal", "missing")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

// --- Lockfile ---

func TestMemoryRepo_Lockfile(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	// No lockfile initially.
	_, err := repo.ReadLockfile(ctx)
	require.ErrorIs(t, err, dots.ErrNotExist)

	lockfile := &dots.Lockfile{
		State: dots.LockfileState{
			ActiveProfile: "work",
			LastApplied:   time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC),
			Platform:      "darwin-arm64",
			LinkStrategy:  dots.LinkSymlink,
		},
		Installed: []dots.InstalledPackage{
			{
				Package:      "personal/nvim",
				Tap:          "personal",
				Commit:       "abc123",
				Version:      "1.0.0",
				Type:         "base",
				LinkStrategy: dots.LinkSymlink,
				Files: []dots.InstalledFile{
					{Src: "init.lua", Dest: "~/.config/nvim/init.lua", Origin: "base", Method: "symlink"},
				},
			},
		},
	}

	err = repo.WriteLockfile(ctx, lockfile)
	require.NoError(t, err)

	got, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	require.Equal(t, "work", got.State.ActiveProfile)
	require.Len(t, got.Installed, 1)
	require.Equal(t, "personal/nvim", got.Installed[0].Package)
}

// --- Backups ---

func TestMemoryRepo_Backups(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	// No backups initially.
	paths, err := repo.ListBackups(ctx)
	require.NoError(t, err)
	require.Empty(t, paths)

	// Backup a file.
	data := []byte("original content")
	err = repo.BackupFile(ctx, "/home/user/.gitconfig", data)
	require.NoError(t, err)

	// List backups.
	paths, err = repo.ListBackups(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"/home/user/.gitconfig"}, paths)

	// Restore.
	restored, err := repo.RestoreFile(ctx, "/home/user/.gitconfig")
	require.NoError(t, err)
	require.Equal(t, data, restored)
}

func TestMemoryRepo_RestoreFile_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	_, err := repo.RestoreFile(ctx, "/missing")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

// --- Mutation isolation ---

func TestMemoryRepo_LockfileMutationIsolation(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	lockfile := &dots.Lockfile{
		State: dots.LockfileState{ActiveProfile: "work"},
		Installed: []dots.InstalledPackage{
			{Package: "personal/nvim"},
		},
	}
	err := repo.WriteLockfile(ctx, lockfile)
	require.NoError(t, err)

	// Mutate the original — should not affect stored copy.
	lockfile.State.ActiveProfile = "mutated"
	lockfile.Installed[0].Package = "mutated"

	got, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	require.Equal(t, "work", got.State.ActiveProfile)
	require.Equal(t, "personal/nvim", got.Installed[0].Package)
}
