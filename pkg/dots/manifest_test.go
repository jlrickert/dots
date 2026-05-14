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
	require.Equal(t, "@config/nvim/init.lua", m.Links["init.lua"].Target)
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
	require.Equal(t, "@config/nvim/init.lua", r.Links["init.lua"].Target)
	require.Equal(t, "@config/nvim/lua/", r.Links["lua/"].Target)
	require.Equal(t, "@config/nvim/after/", r.Links["after/"].Target)
	// String shorthand parses with mode=auto.
	require.Equal(t, dots.LinkModeAuto, r.Links["lua/"].Mode)

	// Darwin OS link merged.
	require.Equal(t, "@config/nvim/lua/clipboard.lua", r.Links["helpers/mac-clipboard.lua"].Target)

	// Darwin-arm64 link merged.
	require.Equal(t, "@bin/nvim-silicon", r.Links["bin/nvim-silicon-arm"].Target)

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
	require.Equal(t, "@config/nvim/init.lua", r.Links["init.lua"].Target)

	// Windows link.
	require.Equal(t, "@config/nvim/lua/clipboard.lua", r.Links["helpers/win-clipboard.lua"].Target)

	// No darwin or linux links.
	_, hasMac := r.Links["helpers/mac-clipboard.lua"]
	require.False(t, hasMac)
	_, hasXclip := r.Links["helpers/xclip.lua"]
	require.False(t, hasXclip)

	// Windows hook.
	require.Equal(t, "scripts/install-plugins.ps1", r.Hooks.PostInstall)
}

func TestResolveManifest_LinuxAmd64(t *testing.T) {
	m, err := dots.ParseManifest([]byte(nvimManifest))
	require.NoError(t, err)

	p := dots.Platform{OS: "linux", Arch: "amd64"}
	r := dots.ResolveManifest(m, p)

	require.Equal(t, "@config/nvim/lua/clipboard.lua", r.Links["helpers/xclip.lua"].Target)
	_, hasMac := r.Links["helpers/mac-clipboard.lua"]
	require.False(t, hasMac)
	// Hook stays as base (linux has no hook override).
	require.Equal(t, "scripts/install-plugins.sh", r.Hooks.PostInstall)
}

func TestResolveSelfRefs_Requires(t *testing.T) {
	data := []byte(`
package:
  name: bash
  requires:
    - "@self/common-shell"
    - personal/zsh

platform:
  darwin:
    requires:
      - "@self/mac-only"
      - external/brew
`)
	m, err := dots.ParseManifest(data)
	require.NoError(t, err)

	r := dots.ResolveManifest(m, dots.Platform{OS: "darwin", Arch: "arm64"})
	require.NoError(t, r.ResolveSelfRefs("jared"))

	require.Equal(t, []string{
		"jared/common-shell",
		"personal/zsh",
		"jared/mac-only",
		"external/brew",
	}, r.Package.Requires)
}

func TestResolveSelfRefs_OverlayBase(t *testing.T) {
	data := []byte(`
package:
  name: bash-work

overlay:
  base: "@self/bash"
  strategy: append
  priority: 40
`)
	m, err := dots.ParseManifest(data)
	require.NoError(t, err)

	r := dots.ResolveManifest(m, dots.Platform{OS: "linux", Arch: "amd64"})
	require.NoError(t, r.ResolveSelfRefs("jared"))

	require.NotNil(t, r.Overlay)
	require.Equal(t, "jared/bash", r.Overlay.Base)

	// Mutation must not leak into the source manifest.
	require.Equal(t, "@self/bash", m.Overlay.Base)
}

func TestResolveSelfRefs_Idempotent(t *testing.T) {
	data := []byte(`
package:
  name: bash
  requires: ["@self/common-shell", "other/pkg"]
`)
	m, err := dots.ParseManifest(data)
	require.NoError(t, err)

	r := dots.ResolveManifest(m, dots.Platform{OS: "linux", Arch: "amd64"})
	require.NoError(t, r.ResolveSelfRefs("jared"))
	require.NoError(t, r.ResolveSelfRefs("jared"))

	require.Equal(t, []string{"jared/common-shell", "other/pkg"}, r.Package.Requires)
}

func TestResolveSelfRef_EmptyTap(t *testing.T) {
	// Non-self refs pass through even when currentTap is empty.
	ref, err := dots.ResolveSelfRef("personal/zsh", "")
	require.NoError(t, err)
	require.Equal(t, "personal/zsh", ref)

	// Self refs require a current tap.
	_, err = dots.ResolveSelfRef("@self/common-shell", "")
	require.ErrorIs(t, err, dots.ErrParse)
}

func TestResolveSelfRef_NoPrefix(t *testing.T) {
	ref, err := dots.ResolveSelfRef("personal/zsh", "jared")
	require.NoError(t, err)
	require.Equal(t, "personal/zsh", ref)
}

func TestParseManifest_LinkSpec_StringShorthand(t *testing.T) {
	data := []byte(`
package:
  name: nvim
links:
  init.lua: "@config/nvim/init.lua"
  lua/: "@config/nvim/lua/"
`)
	m, err := dots.ParseManifest(data)
	require.NoError(t, err)

	// String shorthand parses with mode=auto and no excludes.
	require.Equal(t, "@config/nvim/init.lua", m.Links["init.lua"].Target)
	require.Equal(t, dots.LinkModeAuto, m.Links["init.lua"].Mode)
	require.Empty(t, m.Links["init.lua"].Exclude)

	require.Equal(t, "@config/nvim/lua/", m.Links["lua/"].Target)
	require.Equal(t, dots.LinkModeAuto, m.Links["lua/"].Mode)
}

func TestParseManifest_LinkSpec_ObjectForm(t *testing.T) {
	data := []byte(`
package:
  name: poststone
links:
  factorizers/:
    target: "@config/poststone/factorizers/"
    mode: copy
    exclude: ["__pycache__", "*.pyc"]
`)
	m, err := dots.ParseManifest(data)
	require.NoError(t, err)

	spec := m.Links["factorizers/"]
	require.Equal(t, "@config/poststone/factorizers/", spec.Target)
	require.Equal(t, dots.LinkModeCopy, spec.Mode)
	require.Equal(t, []string{"__pycache__", "*.pyc"}, spec.Exclude)
}

func TestParseManifest_LinkSpec_RejectsUnknownMode(t *testing.T) {
	data := []byte(`
package:
  name: nvim
links:
  lua/:
    target: "@config/nvim/lua/"
    mode: hardlink
`)
	_, err := dots.ParseManifest(data)
	// hardlink isn't a directory mode — must reject.
	require.ErrorIs(t, err, dots.ErrParse)
}

func TestParseManifest_LinkSpec_RequiresTarget(t *testing.T) {
	data := []byte(`
package:
  name: nvim
links:
  lua/:
    mode: copy
`)
	_, err := dots.ParseManifest(data)
	require.ErrorIs(t, err, dots.ErrParse)
}

func TestResolveManifest_LinkSpec_PlatformCascade(t *testing.T) {
	data := []byte(`
package:
  name: nvim
links:
  shared/: "@config/nvim/shared/"

platform:
  darwin:
    links:
      darwin-only/:
        target: "@config/nvim/darwin/"
        mode: copy
        exclude: [".DS_Store"]
`)
	m, err := dots.ParseManifest(data)
	require.NoError(t, err)

	r := dots.ResolveManifest(m, dots.Platform{OS: "darwin", Arch: "arm64"})

	// Base directory entry preserved as auto-mode shorthand.
	require.Equal(t, "@config/nvim/shared/", r.Links["shared/"].Target)
	require.Equal(t, dots.LinkModeAuto, r.Links["shared/"].Mode)

	// Platform-specific object-form entry merged in with mode + excludes.
	darwin := r.Links["darwin-only/"]
	require.Equal(t, "@config/nvim/darwin/", darwin.Target)
	require.Equal(t, dots.LinkModeCopy, darwin.Mode)
	require.Equal(t, []string{".DS_Store"}, darwin.Exclude)
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
