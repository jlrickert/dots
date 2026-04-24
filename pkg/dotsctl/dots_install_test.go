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

func TestInstall_DefaultStrategyIsCopy(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedPackage(t, d, repo, "personal", "testpkg")

	result, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/testpkg"})
	require.NoError(t, err)
	require.Equal(t, "copy", result.Files[0].Method)
}

// seedWorkModePackage writes a package to a real filesystem path and enables
// work_mode for the tap pointing at it. Mirrors what `dots work on` produces in
// production: a tap pointing at a user checkout that lives outside dots' own
// state directory.
func seedWorkModePackage(t *testing.T, d *dotsctl.Dots, repo *dots.MemoryRepo, tap, pkg string, manifest []byte) string {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, repo.AddTap(ctx, dots.TapInfo{Name: tap, URL: "test"}))

	workDir := filepath.Join(t.TempDir(), "checkout")
	pkgDir := filepath.Join(workDir, pkg)
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "Dotfile.yaml"), manifest, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "testfile"), []byte("content"), 0o644))
	require.NoError(t, repo.AddPackage(tap, dots.PackageInfo{Tap: tap, Name: pkg, Dir: pkg}, manifest))

	cfg, err := d.ConfigService.Config(false)
	require.NoError(t, err)
	if cfg.WorkMode == nil {
		cfg.WorkMode = map[string]string{}
	}
	cfg.WorkMode[tap] = workDir
	require.NoError(t, d.ConfigService.Save(cfg))

	return pkgDir
}

func TestInstall_WorkModeUsesSymlink(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	manifest := []byte("package:\n  name: testpkg\nlinks:\n  testfile: \"@config/testpkg/testfile\"\n")
	seedWorkModePackage(t, d, repo, "personal", "testpkg", manifest)

	result, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/testpkg"})
	require.NoError(t, err)
	require.Equal(t, "symlink", result.Files[0].Method)
}

func TestInstall_ManifestOverridesWorkMode(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	// Manifest forces copy — should win over the work-mode default of symlink.
	manifest := []byte("package:\n  name: testpkg\n  link_strategy: copy\nlinks:\n  testfile: \"@config/testpkg/testfile\"\n")
	seedWorkModePackage(t, d, repo, "personal", "testpkg", manifest)

	result, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/testpkg"})
	require.NoError(t, err)
	require.Equal(t, "copy", result.Files[0].Method, "manifest link_strategy must win over work-mode default")
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

func TestInstall_NonEmptyDirAtDest(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedPackage(t, d, repo, "personal", "testpkg")

	dry, err := d.Install(ctx, dotsctl.InstallOptions{
		Package: "personal/testpkg",
		DryRun:  true,
	})
	require.NoError(t, err)
	require.Len(t, dry.Files, 1)
	dest := dry.Files[0].Dest

	// Seed the user's data at the destination through Runtime so it lands
	// where prepareDest will inspect it. Prior to the fix, os.Remove
	// silently failed on the non-empty directory and PlaceLink reported
	// a confusing "file exists" error.
	require.NoError(t, d.Runtime.Mkdir(dest, 0o755, true))
	require.NoError(t, d.Runtime.WriteFile(filepath.Join(dest, "user-data"), []byte("keep me"), 0o644))

	_, err = d.Install(ctx, dotsctl.InstallOptions{Package: "personal/testpkg"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-empty directory")

	// User data must still be readable — the failed install must not have
	// deleted anything at the destination.
	data, err := d.Runtime.ReadFile(filepath.Join(dest, "user-data"))
	require.NoError(t, err)
	require.Equal(t, []byte("keep me"), data)
}

func TestInstall_EmptyDirAtDest(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedPackage(t, d, repo, "personal", "testpkg")

	dry, err := d.Install(ctx, dotsctl.InstallOptions{
		Package: "personal/testpkg",
		DryRun:  true,
	})
	require.NoError(t, err)
	dest := dry.Files[0].Dest

	require.NoError(t, d.Runtime.Mkdir(dest, 0o755, true))

	_, err = d.Install(ctx, dotsctl.InstallOptions{Package: "personal/testpkg"})
	require.NoError(t, err)

	// After install, dest must no longer be a directory — prepareDest
	// should have cleared the empty dir and PlaceLink should have
	// produced the package's link/copy.
	info, err := d.Runtime.Stat(dest, false)
	if err == nil {
		require.False(t, info.IsDir(), "empty dir should have been cleared and replaced")
	}
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
