package dotsctl_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/stretchr/testify/require"
)

func seedPackage(t *testing.T, d *dotsctl.Dots, repo *dots.MemoryRepo, tap, pkg string) string {
	t.Helper()
	ctx := context.Background()

	// Register tap if not exists
	_ = repo.AddTap(ctx, dots.TapInfo{Name: tap, URL: "test"})

	// Create package directory with source files
	pkgDir := d.PathService.TapsDir() + "/" + tap + "/" + pkg
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))

	manifest := []byte("package:\n  name: " + pkg + "\nlinks:\n  testfile: \"@config/" + pkg + "/testfile\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "Dotfile.yaml"), manifest, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "testfile"), []byte("content from "+pkg), 0o644))

	// Also register in MemoryRepo
	require.NoError(t, repo.AddPackage(tap, dots.PackageInfo{Tap: tap, Name: pkg, Dir: pkg}, manifest))

	return pkgDir
}

func TestInstall_Basic(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedPackage(t, d, repo, "personal", "testpkg")

	result, err := d.Install(ctx, dotsctl.InstallOptions{
		Package: "personal/testpkg",
	})
	require.NoError(t, err)
	require.Equal(t, "personal/testpkg", result.Package)
	require.Len(t, result.Files, 1)
	require.False(t, result.DryRun)

	// Verify link was created
	destFile := result.Files[0].Dest
	_, err = os.Lstat(destFile)
	require.NoError(t, err)
}

func TestInstall_DryRun(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedPackage(t, d, repo, "personal", "testpkg")

	result, err := d.Install(ctx, dotsctl.InstallOptions{
		Package: "personal/testpkg",
		DryRun:  true,
	})
	require.NoError(t, err)
	require.True(t, result.DryRun)
	require.Len(t, result.Files, 1)

	// Should NOT have created the link
	_, err = os.Lstat(result.Files[0].Dest)
	require.True(t, os.IsNotExist(err))
}

func TestInstall_RecordsInLockfile(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedPackage(t, d, repo, "personal", "testpkg")

	_, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/testpkg"})
	require.NoError(t, err)

	lockfile, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	require.Len(t, lockfile.Installed, 1)
	require.Equal(t, "personal/testpkg", lockfile.Installed[0].Package)
}

func TestInstall_InvalidRef(t *testing.T) {
	d, _ := newTestDots(t)
	ctx := context.Background()

	_, err := d.Install(ctx, dotsctl.InstallOptions{Package: "badref"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid package reference")
}

func TestInstall_CopyStrategy(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedPackage(t, d, repo, "personal", "testpkg")

	result, err := d.Install(ctx, dotsctl.InstallOptions{
		Package:  "personal/testpkg",
		Strategy: dots.LinkCopy,
	})
	require.NoError(t, err)
	require.Equal(t, "copy", result.Files[0].Method)

	// Verify it's a copy, not a symlink
	info, err := os.Lstat(result.Files[0].Dest)
	require.NoError(t, err)
	require.False(t, info.Mode()&os.ModeSymlink != 0)
}

// --- Remove ---

func TestRemove_Basic(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedPackage(t, d, repo, "personal", "testpkg")

	result, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/testpkg"})
	require.NoError(t, err)

	destFile := result.Files[0].Dest

	err = d.Remove(ctx, dotsctl.RemoveOptions{Package: "personal/testpkg"})
	require.NoError(t, err)

	// Link should be removed
	_, err = os.Lstat(destFile)
	require.True(t, os.IsNotExist(err))

	// Should be removed from lockfile
	lockfile, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	require.Empty(t, lockfile.Installed)
}

func TestRemove_NotInstalled(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	// Write an empty lockfile
	require.NoError(t, repo.WriteLockfile(ctx, &dots.Lockfile{}))

	err := d.Remove(ctx, dotsctl.RemoveOptions{Package: "personal/missing"})
	require.ErrorIs(t, err, dots.ErrNotExist)
}

// --- Reinstall ---

func TestReinstall(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedPackage(t, d, repo, "personal", "testpkg")

	_, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/testpkg"})
	require.NoError(t, err)

	result, err := d.Reinstall(ctx, dotsctl.ReinstallOptions{Package: "personal/testpkg"})
	require.NoError(t, err)
	require.Equal(t, "personal/testpkg", result.Package)

	lockfile, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	require.Len(t, lockfile.Installed, 1)
}

// --- Upgrade ---

func TestUpgrade_Single(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedPackage(t, d, repo, "personal", "testpkg")

	_, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/testpkg"})
	require.NoError(t, err)

	err = d.Upgrade(ctx, dotsctl.UpgradeOptions{Package: "personal/testpkg"})
	require.NoError(t, err)

	lockfile, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	require.Len(t, lockfile.Installed, 1)
}
