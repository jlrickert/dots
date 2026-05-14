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

	require.NoError(t, d.WorkStateService.Set(tap, workDir))

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

	// Seed the user's data at the destination on the real filesystem so
	// it lands where prepareDest (raw os.Lstat) and PlaceLink (raw
	// os.Symlink) both look. The dry-run dest is already a jail-prefixed
	// absolute path; routing through Runtime.Mkdir would re-jail and
	// write to a different location than prepareDest inspects.
	require.NoError(t, os.MkdirAll(dest, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "user-data"), []byte("keep me"), 0o644))

	_, err = d.Install(ctx, dotsctl.InstallOptions{Package: "personal/testpkg"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-empty directory")

	// User data must still be readable — the failed install must not have
	// deleted anything at the destination.
	data, err := os.ReadFile(filepath.Join(dest, "user-data"))
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

	require.NoError(t, os.MkdirAll(dest, 0o755))

	_, err = d.Install(ctx, dotsctl.InstallOptions{Package: "personal/testpkg"})
	require.NoError(t, err)

	// After install, dest must no longer be a directory — prepareDest
	// should have cleared the empty dir and PlaceLink should have
	// produced the package's link/copy.
	info, err := os.Lstat(dest)
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

// seedDirPackage stages a package whose Dotfile.yaml mixes a file link, a
// directory-symlink shorthand, and an object-form directory copy with an
// __pycache__ exclude. Returns the package directory so tests can poke at
// the source tree if needed.
func seedDirPackage(t *testing.T, d *dotsctl.Dots, repo *dots.MemoryRepo, tap, pkg string) string {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, repo.AddTap(ctx, dots.TapInfo{Name: tap, URL: "test"}))

	pkgDir := filepath.Join(d.PathService.TapsDir(), tap, pkg)
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))

	manifest := []byte(`package:
  name: ` + pkg + `
  link_strategy: symlink
links:
  init.lua: "@config/` + pkg + `/init.lua"
  lua/: "@config/` + pkg + `/lua/"
  factorizers/:
    target: "@config/` + pkg + `/factorizers/"
    mode: copy
    exclude: ["__pycache__"]
`)
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "Dotfile.yaml"), manifest, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "init.lua"), []byte("-- init"), 0o644))

	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "lua"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "lua", "core.lua"), []byte("-- core"), 0o644))

	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "factorizers", "__pycache__"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "factorizers", "__pycache__", "x.pyc"), []byte("cache"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "factorizers", "main.py"), []byte("main"), 0o644))

	require.NoError(t, repo.AddPackage(tap, dots.PackageInfo{Tap: tap, Name: pkg, Dir: pkg}, manifest))
	return pkgDir
}

func TestInstall_DirectoryLinks_EndToEnd(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedDirPackage(t, d, repo, "personal", "appdir")

	result, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/appdir"})
	require.NoError(t, err)

	// Three logical entries (file + dir-symlink + dir-copy w/ excludes).
	// The dir-copy emits exactly one leaf (main.py); __pycache__ is excluded.
	methods := map[string]int{}
	for _, f := range result.Files {
		methods[f.Method]++
	}
	require.Equal(t, 1, methods["symlink"], "init.lua → symlink")
	require.Equal(t, 1, methods["symlink-dir"], "lua/ → symlink-dir")
	require.Equal(t, 1, methods["copy-dir-leaf"], "factorizers/main.py → copy-dir-leaf")
	require.Len(t, result.Files, 3)

	// __pycache__ contents must not have been copied.
	for _, f := range result.Files {
		require.NotContains(t, f.Dest, "__pycache__")
	}
}

// seedSymlinkDirPackage stages a minimal package whose only directory
// link is a single symlink-dir entry. Isolated from the copy-dir-leaf
// shape so reinstall tests target prepareDest's symlink branch alone.
func seedSymlinkDirPackage(t *testing.T, d *dotsctl.Dots, repo *dots.MemoryRepo, tap, pkg string) string {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, repo.AddTap(ctx, dots.TapInfo{Name: tap, URL: "test"}))

	pkgDir := filepath.Join(d.PathService.TapsDir(), tap, pkg)
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))

	manifest := []byte(`package:
  name: ` + pkg + `
  link_strategy: symlink
links:
  lua/: "@config/` + pkg + `/lua/"
`)
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "Dotfile.yaml"), manifest, 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "lua"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "lua", "core.lua"), []byte("-- core"), 0o644))

	require.NoError(t, repo.AddPackage(tap, dots.PackageInfo{Tap: tap, Name: pkg, Dir: pkg}, manifest))
	return pkgDir
}

// TestInstall_SymlinkDir_ReinstallIsIdempotent guards the upgrade-path
// hazard fixed in prepareDest: a second `dots install` of a package that
// placed a symlink-dir on the first run must not follow that symlink back
// into the source package. Before the fix, prepareDest used Runtime.Stat
// without the Lstat-style discriminator path, so a prior dir-symlink at
// dest was either followed (ReadDir into source) or — once Lstat was
// added — the symlink branch had to short-circuit before IsDir to avoid
// a spurious "non-empty directory at dest" abort.
func TestInstall_SymlinkDir_ReinstallIsIdempotent(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	pkgDir := seedSymlinkDirPackage(t, d, repo, "personal", "appdir")

	first, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/appdir"})
	require.NoError(t, err)
	require.Len(t, first.Files, 1)
	require.Equal(t, "symlink-dir", first.Files[0].Method)

	symlinkDirDest := first.Files[0].Dest
	symlinkDirSrc := first.Files[0].Src
	require.Equal(t, filepath.Join(pkgDir, "lua"), symlinkDirSrc,
		"sanity: symlink-dir should target the source lua/ inside the package")

	// Re-run install. This is the upgrade-path shape: dest is already a
	// dir-symlink that points into the source package, and a naive Stat
	// would see the source dir as non-empty (or worse, ReadDir into it).
	second, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/appdir"})
	require.NoError(t, err, "reinstall over an existing symlink-dir must succeed")

	// The dest must still be a symlink (not a regular dir, not gone) and
	// must still point at the source dir.
	info, err := os.Lstat(symlinkDirDest)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&os.ModeSymlink,
		"dest must remain a symlink after reinstall, not be replaced by a real dir")
	target, err := os.Readlink(symlinkDirDest)
	require.NoError(t, err)
	require.Equal(t, symlinkDirSrc, target,
		"symlink target must still point at the source dir, not stomped into source")

	// And the source tree must be untouched — prepareDest must never
	// recurse through the dir-symlink to the source package.
	srcEntry := filepath.Join(symlinkDirSrc, "core.lua")
	srcData, err := os.ReadFile(srcEntry)
	require.NoError(t, err, "source file under symlink target must still exist")
	require.Equal(t, []byte("-- core"), srcData,
		"source file content must be unchanged; install must not have written into source")

	// Lockfile-side: still exactly one symlink-dir row at the same dest.
	require.Len(t, second.Files, 1)
	require.Equal(t, "symlink-dir", second.Files[0].Method)
	require.Equal(t, symlinkDirDest, second.Files[0].Dest)
}

// TestInstall_SymlinkDir_ReinstallAfterSrcEdit_RetainsSymlink covers the
// "source has been edited" branch: a re-install must not consult the
// symlink target's contents — it should remove the prior symlink and
// place a fresh one with the same target. The user's edit under the
// source must be visible through the symlink afterward.
func TestInstall_SymlinkDir_ReinstallAfterSrcEdit_RetainsSymlink(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	pkgDir := seedSymlinkDirPackage(t, d, repo, "personal", "appdir")

	first, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/appdir"})
	require.NoError(t, err)
	require.Len(t, first.Files, 1)
	symlinkDirDest := first.Files[0].Dest

	// Edit the source (as a tap update / user checkout edit would).
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "lua", "core.lua"),
		[]byte("-- core v2"),
		0o644,
	))

	_, err = d.Install(ctx, dotsctl.InstallOptions{Package: "personal/appdir"})
	require.NoError(t, err)

	// Symlink retained, edit visible through it.
	info, err := os.Lstat(symlinkDirDest)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&os.ModeSymlink)

	got, err := os.ReadFile(filepath.Join(symlinkDirDest, "core.lua"))
	require.NoError(t, err)
	require.Equal(t, []byte("-- core v2"), got,
		"edit at the source must be visible through the retained symlink")
}

// TestUpgrade_SymlinkDirPackage exercises the actual call site the
// reviewer flagged: `dots upgrade` on a package whose first install
// placed a symlink-dir. Upgrade does Remove + Install internally, so the
// hazard manifests if Remove leaves the symlink in place and Install's
// prepareDest then has to handle it. Either way, the upgrade must
// succeed and leave the symlink pointing at the (current) source.
func TestUpgrade_SymlinkDirPackage(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	pkgDir := seedSymlinkDirPackage(t, d, repo, "personal", "appdir")

	first, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/appdir"})
	require.NoError(t, err)
	symlinkDirDest := first.Files[0].Dest

	require.NoError(t, d.Upgrade(ctx, dotsctl.UpgradeOptions{Package: "personal/appdir"}))

	info, err := os.Lstat(symlinkDirDest)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&os.ModeSymlink,
		"upgrade must leave a symlink at dest, not a real dir")
	target, err := os.Readlink(symlinkDirDest)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(pkgDir, "lua"), target)
}

func TestRemove_DirectoryLinks_CleansSymlinkAndCopyTree(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedDirPackage(t, d, repo, "personal", "appdir")

	result, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/appdir"})
	require.NoError(t, err)

	// Capture dest paths for later assertions.
	var symlinkDirDest, copyLeafDest string
	for _, f := range result.Files {
		switch f.Method {
		case "symlink-dir":
			symlinkDirDest = f.Dest
		case "copy-dir-leaf":
			copyLeafDest = f.Dest
		}
	}
	require.NotEmpty(t, symlinkDirDest)
	require.NotEmpty(t, copyLeafDest)

	require.NoError(t, d.Remove(ctx, dotsctl.RemoveOptions{Package: "personal/appdir"}))

	// symlink-dir removed as a single op.
	_, err = os.Lstat(symlinkDirDest)
	require.True(t, os.IsNotExist(err))

	// copy-dir-leaf removed and parent directory pruned.
	_, err = os.Lstat(copyLeafDest)
	require.True(t, os.IsNotExist(err))
	_, err = os.Lstat(filepath.Dir(copyLeafDest))
	require.True(t, os.IsNotExist(err), "empty parent should be pruned bottom-up")
}

func TestDiff_FlagsCopyDirLeafDrift(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedDirPackage(t, d, repo, "personal", "appdir")

	result, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/appdir"})
	require.NoError(t, err)

	var copyLeafDest string
	for _, f := range result.Files {
		if f.Method == "copy-dir-leaf" {
			copyLeafDest = f.Dest
			break
		}
	}
	require.NotEmpty(t, copyLeafDest)

	// User edits the copied leaf in place.
	require.NoError(t, os.WriteFile(copyLeafDest, []byte("user-edit"), 0o644))

	diffs, err := d.Diff(ctx, "personal/appdir")
	require.NoError(t, err)
	require.Len(t, diffs, 1)
	require.Equal(t, copyLeafDest, diffs[0].File)
	require.Equal(t, "changed", diffs[0].Status)
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
