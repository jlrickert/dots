package dots_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

func TestMergeFiles_Append(t *testing.T) {
	dir := t.TempDir()
	layerDir := filepath.Join(dir, "overlay")
	require.NoError(t, os.MkdirAll(layerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layerDir, "config"), []byte("overlay line"), 0o644))

	result, err := dots.MergeFiles(
		[]byte("base line"),
		[]dots.OverlayLayer{{
			Package:  "work/extra",
			Priority: 10,
			Strategy: dots.MergeAppend,
			Dir:      layerDir,
		}},
		"config",
	)
	require.NoError(t, err)
	require.Contains(t, string(result), "base line")
	require.Contains(t, string(result), "overlay line")
	require.Contains(t, string(result), "# --- overlay: work/extra ---")
}

func TestMergeFiles_Prepend(t *testing.T) {
	dir := t.TempDir()
	layerDir := filepath.Join(dir, "overlay")
	require.NoError(t, os.MkdirAll(layerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layerDir, "config"), []byte("first"), 0o644))

	result, err := dots.MergeFiles(
		[]byte("second"),
		[]dots.OverlayLayer{{
			Package:  "work/extra",
			Priority: 10,
			Strategy: dots.MergePrepend,
			Dir:      layerDir,
		}},
		"config",
	)
	require.NoError(t, err)
	s := string(result)
	// "first" should appear before "second"
	require.Less(t, indexOf(s, "first"), indexOf(s, "second"))
}

func TestMergeFiles_Replace(t *testing.T) {
	dir := t.TempDir()
	layerDir := filepath.Join(dir, "overlay")
	require.NoError(t, os.MkdirAll(layerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layerDir, "config"), []byte("replaced"), 0o644))

	result, err := dots.MergeFiles(
		[]byte("original"),
		[]dots.OverlayLayer{{
			Package:  "work/extra",
			Priority: 10,
			Strategy: dots.MergeReplace,
			Dir:      layerDir,
		}},
		"config",
	)
	require.NoError(t, err)
	require.Equal(t, "replaced", string(result))
}

func TestMergeFiles_PriorityOrdering(t *testing.T) {
	dir := t.TempDir()

	lowDir := filepath.Join(dir, "low")
	highDir := filepath.Join(dir, "high")
	require.NoError(t, os.MkdirAll(lowDir, 0o755))
	require.NoError(t, os.MkdirAll(highDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(lowDir, "f"), []byte("low"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(highDir, "f"), []byte("high"), 0o644))

	// Replace strategy: highest priority wins (applied last)
	result, err := dots.MergeFiles(
		[]byte("base"),
		[]dots.OverlayLayer{
			{Package: "a/low", Priority: 5, Strategy: dots.MergeReplace, Dir: lowDir},
			{Package: "b/high", Priority: 50, Strategy: dots.MergeReplace, Dir: highDir},
		},
		"f",
	)
	require.NoError(t, err)
	require.Equal(t, "high", string(result))
}

func TestWriteMergedFile(t *testing.T) {
	dir := t.TempDir()
	err := dots.WriteMergedFile(dir, "personal", "nvim", "init.lua", []byte("merged content"))
	require.NoError(t, err)

	path := filepath.Join(dir, "personal", "nvim", "init.lua")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "merged content", string(data))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
