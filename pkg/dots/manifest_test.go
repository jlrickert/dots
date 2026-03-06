package dots_test

import (
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

const nvimManifest = `
package:
  name: nvim
  description: Neovim configuration
  version: 1.2.0
  tags: [editor, neovim]

links:
  init.lua: "@config/nvim/init.lua"
  lua/: "@config/nvim/lua/"
  after/: "@config/nvim/after/"

hooks:
  post_install: scripts/install-plugins.sh

platform:
  darwin:
    links:
      helpers/mac-clipboard.lua: "@config/nvim/lua/clipboard.lua"
    hooks:
      post_install: scripts/install-plugins-mac.sh
    tags: [mac]

  darwin-arm64:
    links:
      bin/nvim-silicon-arm: "@bin/nvim-silicon"

  linux:
    links:
      helpers/xclip.lua: "@config/nvim/lua/clipboard.lua"

  windows:
    links:
      helpers/win-clipboard.lua: "@config/nvim/lua/clipboard.lua"
    hooks:
      post_install: scripts/install-plugins.ps1
`

func TestParseManifest(t *testing.T) {
	m, err := dots.ParseManifest([]byte(nvimManifest))
	require.NoError(t, err)
	require.Equal(t, "nvim", m.Package.Name)
	require.Equal(t, "1.2.0", m.Package.Version)
	require.Equal(t, []string{"editor", "neovim"}, m.Package.Tags)
	require.Equal(t, "@config/nvim/init.lua", m.Links["init.lua"])
	require.Equal(t, "scripts/install-plugins.sh", m.Hooks.PostInstall)
	require.Len(t, m.Platform, 4)
}

func TestParseManifest_MissingName(t *testing.T) {
	data := []byte(`
package:
  description: no name
links:
  foo: bar
`)
	_, err := dots.ParseManifest(data)
	require.ErrorIs(t, err, dots.ErrParse)
}

func TestParseManifest_InvalidYAML(t *testing.T) {
	_, err := dots.ParseManifest([]byte(`{bad`))
	require.ErrorIs(t, err, dots.ErrParse)
}

func TestParseManifest_Overlay(t *testing.T) {
	data := []byte(`
package:
  name: work-nvim
  tags: [work]

overlay:
  base: personal/nvim
  strategy: append
  priority: 50

links:
  work-init.lua: "@config/nvim/lua/work.lua"

merge:
  init.lua: append
`)
	m, err := dots.ParseManifest(data)
	require.NoError(t, err)
	require.NotNil(t, m.Overlay)
	require.Equal(t, "personal/nvim", m.Overlay.Base)
	require.Equal(t, "append", m.Overlay.Strategy)
	require.Equal(t, 50, m.Overlay.Priority)
	require.Equal(t, "append", m.Merge["init.lua"])
}

func TestParseManifest_PlatformRestricted(t *testing.T) {
	data := []byte(`
package:
  name: aerospace
  platforms: [darwin-arm64, darwin-amd64]

links:
  config.toml: "@config/aerospace/config.toml"
`)
	m, err := dots.ParseManifest(data)
	require.NoError(t, err)

	require.True(t, m.SupportsCurrentPlatform(dots.Platform{OS: "darwin", Arch: "arm64"}))
	require.True(t, m.SupportsCurrentPlatform(dots.Platform{OS: "darwin", Arch: "amd64"}))
	require.False(t, m.SupportsCurrentPlatform(dots.Platform{OS: "linux", Arch: "amd64"}))
	require.False(t, m.SupportsCurrentPlatform(dots.Platform{OS: "windows", Arch: "amd64"}))
}

func TestParseManifest_NoPlatformRestriction(t *testing.T) {
	data := []byte(`
package:
  name: git
links:
  .gitconfig: .gitconfig
`)
	m, err := dots.ParseManifest(data)
	require.NoError(t, err)
	require.True(t, m.SupportsCurrentPlatform(dots.Platform{OS: "linux", Arch: "amd64"}))
	require.True(t, m.SupportsCurrentPlatform(dots.Platform{OS: "windows", Arch: "amd64"}))
}

func TestResolveManifest_DarwinArm64(t *testing.T) {
	m, err := dots.ParseManifest([]byte(nvimManifest))
	require.NoError(t, err)

	p := dots.Platform{OS: "darwin", Arch: "arm64"}
	r := dots.ResolveManifest(m, p)

	// Base links preserved.
	require.Equal(t, "@config/nvim/init.lua", r.Links["init.lua"])
	require.Equal(t, "@config/nvim/lua/", r.Links["lua/"])
	require.Equal(t, "@config/nvim/after/", r.Links["after/"])

	// Darwin OS link merged.
	require.Equal(t, "@config/nvim/lua/clipboard.lua", r.Links["helpers/mac-clipboard.lua"])

	// Darwin-arm64 link merged.
	require.Equal(t, "@bin/nvim-silicon", r.Links["bin/nvim-silicon-arm"])

	// Hook replaced by darwin section.
	require.Equal(t, "scripts/install-plugins-mac.sh", r.Hooks.PostInstall)

	// Tags merged.
	require.Contains(t, r.Package.Tags, "editor")
	require.Contains(t, r.Package.Tags, "neovim")
	require.Contains(t, r.Package.Tags, "mac")
}

func TestResolveManifest_WindowsAmd64(t *testing.T) {
	m, err := dots.ParseManifest([]byte(nvimManifest))
	require.NoError(t, err)

	p := dots.Platform{OS: "windows", Arch: "amd64"}
	r := dots.ResolveManifest(m, p)

	// Base links preserved.
	require.Equal(t, "@config/nvim/init.lua", r.Links["init.lua"])

	// Windows link.
	require.Equal(t, "@config/nvim/lua/clipboard.lua", r.Links["helpers/win-clipboard.lua"])

	// No darwin or linux links.
	require.Empty(t, r.Links["helpers/mac-clipboard.lua"])
	require.Empty(t, r.Links["helpers/xclip.lua"])

	// Windows hook.
	require.Equal(t, "scripts/install-plugins.ps1", r.Hooks.PostInstall)
}

func TestResolveManifest_LinuxAmd64(t *testing.T) {
	m, err := dots.ParseManifest([]byte(nvimManifest))
	require.NoError(t, err)

	p := dots.Platform{OS: "linux", Arch: "amd64"}
	r := dots.ResolveManifest(m, p)

	require.Equal(t, "@config/nvim/lua/clipboard.lua", r.Links["helpers/xclip.lua"])
	require.Empty(t, r.Links["helpers/mac-clipboard.lua"])
	// Hook stays as base (linux has no hook override).
	require.Equal(t, "scripts/install-plugins.sh", r.Hooks.PostInstall)
}

func TestResolveManifest_LinkStrategyOverride(t *testing.T) {
	data := []byte(`
package:
  name: ssh
  link_strategy: copy

links:
  config: "@config/ssh/config"

platform:
  windows:
    link_strategy: hardlink
`)
	m, err := dots.ParseManifest(data)
	require.NoError(t, err)

	t.Run("base strategy", func(t *testing.T) {
		r := dots.ResolveManifest(m, dots.Platform{OS: "linux", Arch: "amd64"})
		require.Equal(t, dots.LinkCopy, r.LinkStrategy)
	})

	t.Run("platform override", func(t *testing.T) {
		r := dots.ResolveManifest(m, dots.Platform{OS: "windows", Arch: "amd64"})
		require.Equal(t, dots.LinkHardlink, r.LinkStrategy)
	})
}
