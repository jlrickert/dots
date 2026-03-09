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

	result, err := dots.PlaceLink(dots.LinkAction{
		Src:      src,
		Dest:     dest,
		Strategy: dots.LinkSymlink,
	})
	require.NoError(t, err)
	require.Equal(t, "symlink", result.Method)

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

	result, err := dots.PlaceLink(dots.LinkAction{
		Src:      src,
		Dest:     dest,
		Strategy: dots.LinkCopy,
	})
	require.NoError(t, err)
	require.Equal(t, "copy", result.Method)
	require.NotEmpty(t, result.Checksum)

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

	result, err := dots.PlaceLink(dots.LinkAction{
		Src:      src,
		Dest:     dest,
		Strategy: dots.LinkHardlink,
	})
	require.NoError(t, err)
	require.Equal(t, "hardlink", result.Method)

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

func TestResolveLinkActions(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "tap", "nvim")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))

	resolver := dots.NewAliasResolver(
		dots.Platform{OS: "linux", Arch: "amd64"},
		func(key string) string {
			if key == "HOME" {
				return filepath.Join(dir, "home")
			}
			return ""
		},
	)

	resolved := &dots.ResolvedManifest{
		Links: map[string]string{
			"init.lua": "@config/nvim/init.lua",
		},
	}

	actions, err := dots.ResolveLinkActions(resolved, pkgDir, resolver, dots.LinkSymlink)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	require.Equal(t, filepath.Join(pkgDir, "init.lua"), actions[0].Src)
	require.Contains(t, actions[0].Dest, "nvim/init.lua")
	require.Equal(t, dots.LinkSymlink, actions[0].Strategy)
}
