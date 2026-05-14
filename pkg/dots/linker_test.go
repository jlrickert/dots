package dots_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

func TestPlaceLink_Symlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "init.lua")
	dest := filepath.Join(dir, "dest", "nvim", "init.lua")

	require.NoError(t, os.MkdirAll(filepath.Dir(src), 0o755))
	require.NoError(t, os.WriteFile(src, []byte("-- nvim config"), 0o644))

	results, err := dots.PlaceLink(dots.LinkAction{
		Src:      src,
		Dest:     dest,
		Strategy: dots.LinkSymlink,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "symlink", results[0].Method)

	target, err := os.Readlink(dest)
	require.NoError(t, err)
	require.Equal(t, src, target)
}

func TestPlaceLink_Copy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "config")
	dest := filepath.Join(dir, "dest", "config")

	require.NoError(t, os.MkdirAll(filepath.Dir(src), 0o755))
	require.NoError(t, os.WriteFile(src, []byte("content"), 0o644))

	results, err := dots.PlaceLink(dots.LinkAction{
		Src:      src,
		Dest:     dest,
		Strategy: dots.LinkCopy,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "copy", results[0].Method)
	require.NotEmpty(t, results[0].Checksum)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, []byte("content"), data)
}

func TestPlaceLink_Hardlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "file")
	dest := filepath.Join(dir, "dest", "file")

	require.NoError(t, os.MkdirAll(filepath.Dir(src), 0o755))
	require.NoError(t, os.WriteFile(src, []byte("hardlink test"), 0o644))

	results, err := dots.PlaceLink(dots.LinkAction{
		Src:      src,
		Dest:     dest,
		Strategy: dots.LinkHardlink,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "hardlink", results[0].Method)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, []byte("hardlink test"), data)

	// Verify same inode
	srcInfo, _ := os.Stat(src)
	destInfo, _ := os.Stat(dest)
	require.True(t, os.SameFile(srcInfo, destInfo))
}

func TestRemoveLink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))

	err := dots.RemoveLink(path)
	require.NoError(t, err)

	_, err = os.Stat(path)
	require.True(t, os.IsNotExist(err))
}

func TestFileChecksum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(path, []byte("test content"), 0o644))

	checksum, err := dots.FileChecksum(path)
	require.NoError(t, err)
	require.True(t, len(checksum) > 7)
	require.Equal(t, "sha256:", checksum[:7])
}

func TestPlaceLink_DirSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "lua")
	dest := filepath.Join(dir, "dest", "nvim", "lua")

	// Populate the source directory; placeDirSymlink should not enter it.
	require.NoError(t, os.MkdirAll(filepath.Join(src, "subdir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "init.lua"), []byte("-- init"), 0o644))

	results, err := dots.PlaceLink(dots.LinkAction{
		Src:      src,
		Dest:     dest,
		Strategy: dots.LinkSymlink,
		Mode:     dots.LinkModeSymlink,
		IsDir:    true,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "symlink-dir", results[0].Method)

	target, err := os.Readlink(dest)
	require.NoError(t, err)
	require.Equal(t, src, target)
}

func TestPlaceLink_DirCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "factorizers")
	dest := filepath.Join(dir, "dest", "factorizers")

	require.NoError(t, os.MkdirAll(filepath.Join(src, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.py"), []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "b.py"), []byte("b"), 0o644))

	results, err := dots.PlaceLink(dots.LinkAction{
		Src:      src,
		Dest:     dest,
		Strategy: dots.LinkCopy,
		Mode:     dots.LinkModeCopy,
		IsDir:    true,
	})
	require.NoError(t, err)
	require.Len(t, results, 2, "should emit one InstalledFile per regular file")

	// Each result has a checksum and copy-dir-leaf method.
	for _, r := range results {
		require.Equal(t, "copy-dir-leaf", r.Method)
		require.NotEmpty(t, r.Checksum)
	}

	// Files actually copied.
	a, err := os.ReadFile(filepath.Join(dest, "a.py"))
	require.NoError(t, err)
	require.Equal(t, []byte("a"), a)
	b, err := os.ReadFile(filepath.Join(dest, "nested", "b.py"))
	require.NoError(t, err)
	require.Equal(t, []byte("b"), b)
}

func TestPlaceLink_DirCopy_HonorsExclude(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "pkg")
	dest := filepath.Join(dir, "dest", "pkg")

	require.NoError(t, os.MkdirAll(filepath.Join(src, "__pycache__"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "nested", "__pycache__"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "__pycache__", "x.pyc"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "__pycache__", "y.pyc"), []byte("y"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "main.py"), []byte("main"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "lib.py"), []byte("lib"), 0o644))

	results, err := dots.PlaceLink(dots.LinkAction{
		Src:      src,
		Dest:     dest,
		Strategy: dots.LinkCopy,
		Mode:     dots.LinkModeCopy,
		Exclude:  []string{"__pycache__"},
		IsDir:    true,
	})
	require.NoError(t, err)
	require.Len(t, results, 2, "exclude should drop both top-level and nested __pycache__ trees")

	// __pycache__ folders should be absent from dest.
	_, err = os.Stat(filepath.Join(dest, "__pycache__"))
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(dest, "nested", "__pycache__"))
	require.True(t, os.IsNotExist(err))

	// main.py and nested/lib.py should be present.
	_, err = os.Stat(filepath.Join(dest, "main.py"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dest, "nested", "lib.py"))
	require.NoError(t, err)
}

func TestPlaceLink_DirHardlinkDegradesToSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "data")
	dest := filepath.Join(dir, "dest", "data")
	require.NoError(t, os.MkdirAll(src, 0o755))

	// LinkModeAuto + LinkHardlink should fall back to symlink-dir with a
	// stderr warning.
	results, err := dots.PlaceLink(dots.LinkAction{
		Src:      src,
		Dest:     dest,
		Strategy: dots.LinkHardlink,
		Mode:     dots.LinkModeAuto,
		IsDir:    true,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "symlink-dir", results[0].Method)
}

func TestPlaceLink_FileSourceUnaffectedByMode(t *testing.T) {
	// File sources collapse to the historical Strategy-driven flow
	// regardless of Mode. Explicit Mode=copy on a file source must not
	// change behavior.
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "init.lua")
	dest := filepath.Join(dir, "dest", "init.lua")
	require.NoError(t, os.MkdirAll(filepath.Dir(src), 0o755))
	require.NoError(t, os.WriteFile(src, []byte("file"), 0o644))

	results, err := dots.PlaceLink(dots.LinkAction{
		Src:      src,
		Dest:     dest,
		Strategy: dots.LinkSymlink,
		Mode:     dots.LinkModeCopy, // ignored for files
		IsDir:    false,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "symlink", results[0].Method, "file sources follow Strategy, not Mode")
}

func TestMatchExclude_SegmentRule(t *testing.T) {
	require.True(t, dots.MatchExclude("__pycache__", []string{"__pycache__"}))
	require.True(t, dots.MatchExclude("nested/__pycache__", []string{"__pycache__"}))
	require.True(t, dots.MatchExclude("a/b/__pycache__/c.pyc", []string{"__pycache__"}))
	require.True(t, dots.MatchExclude("foo.pyc", []string{"*.pyc"}))
	require.False(t, dots.MatchExclude("main.py", []string{"__pycache__"}))
}

func TestResolveLinkActions(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "tap", "nvim")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))

	resolver := dots.NewAliasResolver(
		dots.Platform{OS: "linux", Arch: "amd64"},
		mapEnv{"HOME": filepath.Join(dir, "home")},
	)

	resolved := &dots.ResolvedManifest{
		Links: map[string]dots.LinkSpec{
			"init.lua": {Target: "@config/nvim/init.lua"},
		},
	}

	actions, err := dots.ResolveLinkActions(resolved, pkgDir, resolver, dots.LinkSymlink)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	require.Equal(t, filepath.Join(pkgDir, "init.lua"), actions[0].Src)
	require.Contains(t, actions[0].Dest, "nvim/init.lua")
	require.Equal(t, dots.LinkSymlink, actions[0].Strategy)
}
