package cli_test

import (
	"testing"

	tu "github.com/jlrickert/cli-toolkit/sandbox"
	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

func TestFixture_ConfigLoads(t *testing.T) {
	sb := NewSandbox(t, tu.WithFixture("config", ".config/dots"))
	data := sb.MustReadFile(".config/dots/config.yaml")
	cfg, err := dots.ParseConfig(data)
	require.NoError(t, err)
	require.Equal(t, dots.LinkSymlink, cfg.Core.LinkStrategy)
	require.Contains(t, cfg.Taps, "personal")
	require.Equal(t, "git@github.com:testuser/dotfiles.git", cfg.Taps["personal"].URL)
}

func TestFixture_ManifestLoads(t *testing.T) {
	sb := NewSandbox(t, tu.WithFixture("taps/personal/nvim", "taps/personal/nvim"))
	data := sb.MustReadFile("taps/personal/nvim/Dotfile.yaml")
	m, err := dots.ParseManifest(data)
	require.NoError(t, err)
	require.Equal(t, "nvim", m.Package.Name)
	require.Equal(t, "1.0.0", m.Package.Version)
	require.Contains(t, m.Links, "init.lua")
}

func TestFixture_ManifestResolve(t *testing.T) {
	sb := NewSandbox(t, tu.WithFixture("taps/personal/nvim", "taps/personal/nvim"))
	data := sb.MustReadFile("taps/personal/nvim/Dotfile.yaml")
	m, err := dots.ParseManifest(data)
	require.NoError(t, err)

	p := dots.Platform{OS: "darwin", Arch: "arm64"}
	r := dots.ResolveManifest(m, p)
	require.Equal(t, "@config/nvim/lua/clipboard.lua", r.Links["helpers/mac-clipboard.lua"])
	require.Equal(t, "scripts/install-plugins-mac.sh", r.Hooks.PostInstall)
}

func TestFixture_MultiplePackages(t *testing.T) {
	sb := NewSandbox(t,
		tu.WithFixture("taps/personal/nvim", "taps/personal/nvim"),
		tu.WithFixture("taps/personal/git", "taps/personal/git"),
	)

	nvimData := sb.MustReadFile("taps/personal/nvim/Dotfile.yaml")
	nvim, err := dots.ParseManifest(nvimData)
	require.NoError(t, err)
	require.Equal(t, "nvim", nvim.Package.Name)

	gitData := sb.MustReadFile("taps/personal/git/Dotfile.yaml")
	gitPkg, err := dots.ParseManifest(gitData)
	require.NoError(t, err)
	require.Equal(t, "git", gitPkg.Package.Name)
}
