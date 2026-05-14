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

func findCheck(t *testing.T, checks []dotsctl.DoctorCheck, name string) dotsctl.DoctorCheck {
	t.Helper()
	for _, c := range checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("doctor check %q not found", name)
	return dotsctl.DoctorCheck{}
}

func TestDoctor_WorkStateFile_AbsentReturnsOk(t *testing.T) {
	d, _ := newTestDots(t)
	checks, err := d.Doctor(context.Background())
	require.NoError(t, err)

	c := findCheck(t, checks, "work state file")
	require.Equal(t, "ok", c.Status)
	require.Contains(t, c.Detail, "no work state")
}

func TestDoctor_WorkModeLegacy_FlagsLegacyEntries(t *testing.T) {
	d, _ := newTestDots(t)

	cfg, err := d.ConfigService.Config(false)
	require.NoError(t, err)
	cfg.WorkMode = map[string]string{"legacy": "/tmp/legacy"}
	require.NoError(t, d.ConfigService.Save(cfg))

	checks, err := d.Doctor(context.Background())
	require.NoError(t, err)

	c := findCheck(t, checks, "work mode (legacy)")
	require.Equal(t, "warn", c.Status)
	require.Contains(t, c.Detail, "legacy")
}

func TestDoctor_WorkModeLegacy_SurfacesConfigParseError(t *testing.T) {
	d, _ := newTestDots(t)

	dir := filepath.Dir(d.ConfigService.ConfigPath)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(d.ConfigService.ConfigPath, []byte("{not: valid: yaml: at: all"), 0o644))
	d.ConfigService.InvalidateCache()

	checks, err := d.Doctor(context.Background())
	require.NoError(t, err)

	c := findCheck(t, checks, "work mode (legacy)")
	require.Equal(t, "error", c.Status)
	require.Contains(t, c.Detail, "load config")
}

func TestDoctor_WorkStateConflict_FlagsDifferingPaths(t *testing.T) {
	d, _ := newTestDots(t)

	cfg, err := d.ConfigService.Config(false)
	require.NoError(t, err)
	cfg.WorkMode = map[string]string{"personal": "/tmp/old"}
	require.NoError(t, d.ConfigService.Save(cfg))

	require.NoError(t, d.WorkStateService.Set("personal", "/tmp/new"))

	checks, err := d.Doctor(context.Background())
	require.NoError(t, err)

	c := findCheck(t, checks, "work state conflict")
	require.Equal(t, "error", c.Status)
	require.Contains(t, c.Detail, "personal")
	require.Contains(t, c.Detail, "/tmp/old")
	require.Contains(t, c.Detail, "/tmp/new")
}

func TestDoctor_WorkStateConflict_NoFalsePositiveOnAgreement(t *testing.T) {
	d, _ := newTestDots(t)

	cfg, err := d.ConfigService.Config(false)
	require.NoError(t, err)
	cfg.WorkMode = map[string]string{"personal": "/tmp/same"}
	require.NoError(t, d.ConfigService.Save(cfg))

	require.NoError(t, d.WorkStateService.Set("personal", "/tmp/same"))

	checks, err := d.Doctor(context.Background())
	require.NoError(t, err)

	c := findCheck(t, checks, "work state conflict")
	require.Equal(t, "ok", c.Status)
}

func TestDoctor_MergeConflictMarkers_FlagsMarkers(t *testing.T) {
	d, _ := newTestDots(t)

	dir := filepath.Dir(d.ConfigService.ConfigPath)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	corrupt := []byte("core:\n<<<<<<< HEAD\n  link_strategy: copy\n=======\n  link_strategy: symlink\n>>>>>>> branch\n")
	require.NoError(t, os.WriteFile(d.ConfigService.ConfigPath, corrupt, 0o644))

	checks, err := d.Doctor(context.Background())
	require.NoError(t, err)

	c := findCheck(t, checks, "merge conflict markers")
	require.Equal(t, "error", c.Status)
	require.Contains(t, c.Detail, d.ConfigService.ConfigPath)
}

func TestDoctor_WorkStateOrphan_FlagsUnregisteredTaps(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	require.NoError(t, repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"}))

	require.NoError(t, d.WorkStateService.Set("personal", "/tmp/personal"))
	require.NoError(t, d.WorkStateService.Set("ghost", "/tmp/ghost"))

	cfg, err := d.ConfigService.Config(false)
	require.NoError(t, err)
	cfg.Taps = map[string]dots.TapConfig{"personal": {URL: "url"}}
	require.NoError(t, d.ConfigService.Save(cfg))

	checks, err := d.Doctor(ctx)
	require.NoError(t, err)

	c := findCheck(t, checks, "work state orphan")
	require.Equal(t, "warn", c.Status)
	require.Contains(t, c.Detail, "ghost")
	require.NotContains(t, c.Detail, "personal")
}

func TestDoctor_DirectoryLinks_FlagsDanglingSymlinkDir(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	// Seed a lockfile entry pointing at a symlink-dir whose target doesn't
	// exist on disk. We don't create the symlink itself — os.Stat on a
	// non-existent path is exactly the dangling shape the check audits.
	require.NoError(t, repo.WriteLockfile(ctx, &dots.Lockfile{
		Installed: []dots.InstalledPackage{
			{
				Package: "personal/appdir",
				Tap:     "personal",
				Files: []dots.InstalledFile{
					{
						Src:    "/tmp/missing-source",
						Dest:   "/tmp/dots-doctor-test/dangling-link",
						Method: "symlink-dir",
					},
				},
			},
		},
	}))

	checks, err := d.Doctor(ctx)
	require.NoError(t, err)

	c := findCheck(t, checks, "directory links")
	require.Equal(t, "warn", c.Status)
	require.Contains(t, c.Detail, "dangling symlink-dir")
	require.Contains(t, c.Detail, "personal/appdir")
}

// TestDoctor_DirectoryLinks_FlagsDriftedCopyDirLeaf mirrors the dangling-
// symlink-dir test for the other failure shape `checkDirectoryLinks`
// audits: a copy-dir-leaf whose on-disk sha256 has drifted from the
// recorded checksum. We seed the lockfile + a matching on-disk file, then
// rewrite the file and assert the drift surfaces in the check detail.
func TestDoctor_DirectoryLinks_FlagsDriftedCopyDirLeaf(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	// Place a real file at the recorded dest and capture its checksum so
	// the lockfile entry agrees with on-disk state at seed time.
	leafDest := filepath.Join(t.TempDir(), "appdir-leaf", "main.py")
	require.NoError(t, os.MkdirAll(filepath.Dir(leafDest), 0o755))
	require.NoError(t, os.WriteFile(leafDest, []byte("main"), 0o644))
	cs, err := dots.FileChecksum(leafDest)
	require.NoError(t, err)

	require.NoError(t, repo.WriteLockfile(ctx, &dots.Lockfile{
		Installed: []dots.InstalledPackage{
			{
				Package: "personal/appdir",
				Tap:     "personal",
				Files: []dots.InstalledFile{
					{
						Src:      "/tmp/source/main.py",
						Dest:     leafDest,
						Method:   "copy-dir-leaf",
						Checksum: cs,
					},
				},
			},
		},
	}))

	// Sanity: a doctor run before drift should not flag this entry.
	checks, err := d.Doctor(ctx)
	require.NoError(t, err)
	pre := findCheck(t, checks, "directory links")
	require.Equal(t, "ok", pre.Status,
		"pre-drift state should be clean; got: %s", pre.Detail)

	// User edits the leaf in place — the recorded checksum no longer matches.
	require.NoError(t, os.WriteFile(leafDest, []byte("user-edit"), 0o644))

	checks, err = d.Doctor(ctx)
	require.NoError(t, err)

	c := findCheck(t, checks, "directory links")
	require.Equal(t, "warn", c.Status)
	require.Contains(t, c.Detail, "drifted copy-dir-leaf")
	require.Contains(t, c.Detail, "personal/appdir")
	require.Contains(t, c.Detail, leafDest,
		"detail must identify the drifted leaf path")
}

func TestDoctor_WorkStatePath_FlagsMissingPaths(t *testing.T) {
	d, _ := newTestDots(t)

	require.NoError(t, d.WorkStateService.Set("personal", "/this/path/does/not/exist"))

	checks, err := d.Doctor(context.Background())
	require.NoError(t, err)

	c := findCheck(t, checks, "work state path")
	require.Equal(t, "warn", c.Status)
	require.Contains(t, c.Detail, "personal")
	require.Contains(t, c.Detail, "/this/path/does/not/exist")
}
