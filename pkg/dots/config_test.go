package dots_test

import (
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Minimal(t *testing.T) {
	data := []byte(`
core:
  link_strategy: symlink
`)
	cfg, err := dots.ParseConfig(data)
	require.NoError(t, err)
	require.Equal(t, dots.LinkSymlink, cfg.Core.LinkStrategy)
	// Defaults should be applied.
	require.Equal(t, "main", cfg.Git.DefaultBranch)
	require.Equal(t, "ssh", cfg.Git.Protocol)
}

func TestParseConfig_Full(t *testing.T) {
	data := []byte(`
core:
  active_profile: work # legacy key; ignored now that the lockfile tracks it
  conflict_strategy: overlay
  always_installed: [personal/dots]
  backup: false
  link_strategy: copy

git:
  default_branch: develop
  protocol: https

taps:
  personal:
    url: git@github.com:me/dotfiles.git
    branch: main
    provider: github
    visibility: private
  work:
    url: git@github.com:corp/dotfiles.git

work_mode:
  personal: /home/me/code/dotfiles

aliases:
  "@dots": "@config/dots"
  "@nvim": "@config/nvim"

platform:
  windows:
    link_strategy: copy
  darwin-arm64:
    link_strategy: hardlink
`)
	cfg, err := dots.ParseConfig(data)
	require.NoError(t, err)
	require.Equal(t, []string{"personal/dots"}, cfg.Core.AlwaysInstalled)
	require.Equal(t, "overlay", cfg.Core.ConflictStrategy)
	require.NotNil(t, cfg.Core.Backup)
	require.False(t, *cfg.Core.Backup)
	require.Equal(t, dots.LinkCopy, cfg.Core.LinkStrategy)
	require.Equal(t, "develop", cfg.Git.DefaultBranch)
	require.Equal(t, "https", cfg.Git.Protocol)
	require.Len(t, cfg.Taps, 2)
	require.Equal(t, "git@github.com:me/dotfiles.git", cfg.Taps["personal"].URL)
	require.Equal(t, "/home/me/code/dotfiles", cfg.WorkMode["personal"])
	require.Equal(t, "@config/dots", cfg.Aliases["@dots"])
	require.Equal(t, dots.LinkCopy, cfg.Platform["windows"].LinkStrategy)
	require.Equal(t, dots.LinkHardlink, cfg.Platform["darwin-arm64"].LinkStrategy)
}

func TestParseConfig_Invalid(t *testing.T) {
	data := []byte(`{invalid yaml`)
	_, err := dots.ParseConfig(data)
	require.ErrorIs(t, err, dots.ErrParse)
}

func TestMergeConfig(t *testing.T) {
	base := dots.DefaultConfig()
	base.Core.AlwaysInstalled = []string{"personal/dots"}

	override := &dots.Config{
		Core: dots.CoreConfig{AlwaysInstalled: []string{"personal/zsh", "personal/dots"}},
		Taps: map[string]dots.TapConfig{
			"work": {URL: "git@github.com:corp/dotfiles.git"},
		},
		Aliases: map[string]string{"@work": "@config/work"},
	}

	result := dots.MergeConfig(&base, override)
	require.Equal(t, []string{"personal/dots", "personal/zsh"}, result.Core.AlwaysInstalled)
	require.Equal(t, dots.LinkCopy, result.Core.LinkStrategy) // from base default
	require.Equal(t, "main", result.Git.DefaultBranch)        // from base default
	require.Equal(t, "git@github.com:corp/dotfiles.git", result.Taps["work"].URL)
	require.Equal(t, "@config/work", result.Aliases["@work"])
}

func TestConfig_ResolveCorePlatform(t *testing.T) {
	cfg := dots.DefaultConfig()
	cfg.Platform = map[string]dots.CoreConfig{
		"windows":      {LinkStrategy: dots.LinkCopy},
		"darwin-arm64": {LinkStrategy: dots.LinkHardlink},
	}

	t.Run("no platform match", func(t *testing.T) {
		core := cfg.ResolveCorePlatform(dots.Platform{OS: "linux", Arch: "amd64"})
		require.Equal(t, dots.LinkCopy, core.LinkStrategy)
	})

	t.Run("OS match", func(t *testing.T) {
		core := cfg.ResolveCorePlatform(dots.Platform{OS: "windows", Arch: "amd64"})
		require.Equal(t, dots.LinkCopy, core.LinkStrategy)
	})

	t.Run("OS-arch match overrides OS", func(t *testing.T) {
		core := cfg.ResolveCorePlatform(dots.Platform{OS: "darwin", Arch: "arm64"})
		require.Equal(t, dots.LinkHardlink, core.LinkStrategy)
	})
}

func TestConfig_ResolveCorePlatform_AlwaysInstalledConcatenates(t *testing.T) {
	cfg := dots.DefaultConfig()
	cfg.Core.AlwaysInstalled = []string{"personal/dots"}
	cfg.Platform = map[string]dots.CoreConfig{
		"darwin":       {AlwaysInstalled: []string{"personal/brew", "personal/dots"}},
		"darwin-arm64": {AlwaysInstalled: []string{"personal/rosetta"}},
	}

	core := cfg.ResolveCorePlatform(dots.Platform{OS: "darwin", Arch: "arm64"})
	require.Equal(t, []string{"personal/dots", "personal/brew", "personal/rosetta"}, core.AlwaysInstalled)

	core = cfg.ResolveCorePlatform(dots.Platform{OS: "linux", Arch: "amd64"})
	require.Equal(t, []string{"personal/dots"}, core.AlwaysInstalled)

	// Resolving must not mutate the base list.
	require.Equal(t, []string{"personal/dots"}, cfg.Core.AlwaysInstalled)
}
