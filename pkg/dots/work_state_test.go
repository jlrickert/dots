package dots_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

func TestParseWorkState_Minimal(t *testing.T) {
	data := []byte(`
taps:
  personal: /home/me/code/dotfiles
`)
	state, err := dots.ParseWorkState(data)
	require.NoError(t, err)
	require.Equal(t, "/home/me/code/dotfiles", state.Taps["personal"])
}

func TestParseWorkState_Full(t *testing.T) {
	data := []byte(`
taps:
  personal: /home/me/code/dotfiles
  work: /Users/me/repos/work-dotfiles
  shared: /opt/team/dotfiles
`)
	state, err := dots.ParseWorkState(data)
	require.NoError(t, err)
	require.Len(t, state.Taps, 3)
	require.Equal(t, "/home/me/code/dotfiles", state.Taps["personal"])
	require.Equal(t, "/Users/me/repos/work-dotfiles", state.Taps["work"])
	require.Equal(t, "/opt/team/dotfiles", state.Taps["shared"])
}

func TestParseWorkState_Invalid(t *testing.T) {
	data := []byte(`{not valid yaml`)
	_, err := dots.ParseWorkState(data)
	require.ErrorIs(t, err, dots.ErrParse)
}

func TestParseWorkState_Empty(t *testing.T) {
	state, err := dots.ParseWorkState(nil)
	require.NoError(t, err)
	require.NotNil(t, state.Taps)
	require.Empty(t, state.Taps)
}

func TestLoadWorkStateFile_Missing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.yaml")

	_, err := dots.LoadWorkStateFile(path)
	require.Error(t, err)
	require.True(t, errors.Is(err, os.ErrNotExist))
}

func TestLoadWorkStateFile_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "work.yaml")
	require.NoError(t, os.WriteFile(path, []byte("taps:\n  personal: /tmp/dots\n"), 0o644))

	state, err := dots.LoadWorkStateFile(path)
	require.NoError(t, err)
	require.Equal(t, "/tmp/dots", state.Taps["personal"])
}

func TestDefaultWorkState_NonNilEmpty(t *testing.T) {
	state := dots.DefaultWorkState()
	require.NotNil(t, state.Taps)
	require.Empty(t, state.Taps)
}
