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

func writeProfile(t *testing.T, d *dotsctl.Dots, yaml string) {
	t.Helper()
	require.NoError(t, d.ProfileImport(context.Background(), []byte(yaml)))
}

// writeConfig writes config.yaml with plain os calls because ConfigService
// reads it with os (see seedPackage for the same pattern).
func writeConfig(t *testing.T, d *dotsctl.Dots, yaml string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(d.ConfigService.ConfigPath), 0o755))
	require.NoError(t, os.WriteFile(d.ConfigService.ConfigPath, []byte(yaml), 0o644))
	d.ConfigService.InvalidateCache()
}

func refs(pkgs []dotsctl.ResolvedPackage) []string {
	out := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		out = append(out, p.Ref)
	}
	return out
}

func installedRefs(t *testing.T, repo *dots.MemoryRepo) []string {
	t.Helper()
	lf, err := repo.ReadLockfile(context.Background())
	if err != nil {
		return nil
	}
	var out []string
	for _, p := range lf.Installed {
		out = append(out, p.Package)
	}
	return out
}

// seedDocsExample writes the profiles from the docs/profiles.md example.
func seedDocsExample(t *testing.T, d *dotsctl.Dots) {
	writeProfile(t, d, "name: base\npackages: [personal/zsh, personal/git, personal/tmux]\n")
	writeProfile(t, d, "name: dev\nextends: base\npackages: [personal/nvim, personal/go]\n")
	writeProfile(t, d, "name: work\nextends: [base, dev]\npackages: [work/zellij]\nexclude: [personal/tmux]\n")
}

// --- Resolution ---

func TestResolveProfile_DocsExample(t *testing.T) {
	d, _ := newTestDots(t)
	seedDocsExample(t, d)

	got, err := d.ResolveProfile("work")
	require.NoError(t, err)
	require.Equal(t, []dotsctl.ResolvedPackage{
		{Ref: "personal/zsh", Source: "base"},
		{Ref: "personal/git", Source: "base"},
		{Ref: "personal/nvim", Source: "dev"},
		{Ref: "personal/go", Source: "dev"},
		{Ref: "work/zellij", Source: "work"},
	}, got)
}

func TestResolveProfile_DiamondDedupes(t *testing.T) {
	d, _ := newTestDots(t)
	writeProfile(t, d, "name: base\npackages: [t/a]\n")
	writeProfile(t, d, "name: left\nextends: base\npackages: [t/l]\n")
	writeProfile(t, d, "name: right\nextends: base\npackages: [t/r, t/a]\n")
	writeProfile(t, d, "name: top\nextends: [left, right]\n")

	got, err := d.ResolveProfile("top")
	require.NoError(t, err)
	require.Equal(t, []string{"t/a", "t/l", "t/r"}, refs(got))
}

func TestResolveProfile_Cycle(t *testing.T) {
	d, _ := newTestDots(t)
	writeProfile(t, d, "name: work\nextends: dev\n")
	writeProfile(t, d, "name: dev\nextends: work\n")

	_, err := d.ResolveProfile("work")
	require.ErrorIs(t, err, dots.ErrConflict)
	var cycle *dots.ProfileCycleError
	require.ErrorAs(t, err, &cycle)
	require.Equal(t, []string{"work", "dev", "work"}, cycle.Chain)
}

func TestResolveProfile_SelfCycle(t *testing.T) {
	d, _ := newTestDots(t)
	writeProfile(t, d, "name: loop\nextends: loop\n")

	_, err := d.ResolveProfile("loop")
	var cycle *dots.ProfileCycleError
	require.ErrorAs(t, err, &cycle)
	require.Equal(t, []string{"loop", "loop"}, cycle.Chain)
}

func TestResolveProfile_MissingParent(t *testing.T) {
	d, _ := newTestDots(t)
	writeProfile(t, d, "name: work\nextends: [base]\n")

	_, err := d.ResolveProfile("work")
	require.ErrorIs(t, err, dots.ErrNotExist)
	require.Contains(t, err.Error(), `"base"`)
	require.Contains(t, err.Error(), `extended by "work"`)
}

func TestResolveProfile_MissingProfile(t *testing.T) {
	d, _ := newTestDots(t)
	_, err := d.ResolveProfile("nope")
	require.ErrorIs(t, err, dots.ErrNotExist)
}

func TestResolveProfile_ExcludeOnlyHitsInherited(t *testing.T) {
	d, _ := newTestDots(t)
	writeProfile(t, d, "name: base\npackages: [t/a, t/b]\n")
	writeProfile(t, d, "name: child\nextends: base\npackages: [t/b]\nexclude: [t/a, t/b]\n")

	got, err := d.ResolveProfile("child")
	require.NoError(t, err)
	require.Equal(t, []dotsctl.ResolvedPackage{{Ref: "t/b", Source: "child"}}, got)
}

func TestResolveProfile_ExcludesAreInherited(t *testing.T) {
	d, _ := newTestDots(t)
	writeProfile(t, d, "name: base\npackages: [t/a, t/b]\n")
	writeProfile(t, d, "name: mid\nextends: base\nexclude: [t/a]\n")
	writeProfile(t, d, "name: leaf\nextends: mid\n")

	got, err := d.ResolveProfile("leaf")
	require.NoError(t, err)
	require.Equal(t, []string{"t/b"}, refs(got))
}

// --- Apply ---

func TestProfileApply_Reconciles(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()
	for _, pkg := range []string{"keep", "extra", "manual", "missing"} {
		seedPackage(t, d, repo, "t", pkg)
	}
	for _, pkg := range []string{"keep", "extra", "manual"} {
		_, err := d.Install(ctx, dotsctl.InstallOptions{Package: "t/" + pkg})
		require.NoError(t, err)
	}
	writeProfile(t, d, "name: p\npackages: [t/keep, t/missing]\n")

	before, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	var keepCommit dots.InstalledPackage
	for _, p := range before.Installed {
		if p.Package == "t/keep" {
			keepCommit = p
		}
	}

	result, err := d.ProfileApply(ctx, dotsctl.ProfileApplyOptions{Name: "p"})
	require.NoError(t, err)
	require.Equal(t, []string{"t/extra", "t/manual"}, result.Remove)
	require.Equal(t, []string{"t/missing"}, result.Install)
	require.Equal(t, []string{"t/keep"}, result.Keep)

	require.ElementsMatch(t, []string{"t/keep", "t/missing"}, installedRefs(t, repo))

	lf, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	require.Equal(t, "p", lf.State.ActiveProfile)
	for _, p := range lf.Installed {
		if p.Package == "t/keep" {
			require.Equal(t, keepCommit, p, "kept package must not be reinstalled")
		}
	}
}

func TestProfileApply_DryRunChangesNothing(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()
	seedPackage(t, d, repo, "t", "old")
	seedPackage(t, d, repo, "t", "new")
	_, err := d.Install(ctx, dotsctl.InstallOptions{Package: "t/old"})
	require.NoError(t, err)
	writeProfile(t, d, "name: p\npackages: [t/new]\n")

	result, err := d.ProfileApply(ctx, dotsctl.ProfileApplyOptions{Name: "p", DryRun: true})
	require.NoError(t, err)
	require.True(t, result.DryRun)
	require.Equal(t, []string{"t/old"}, result.Remove)
	require.Equal(t, []string{"t/new"}, result.Install)

	require.Equal(t, []string{"t/old"}, installedRefs(t, repo))
	lf, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	require.Empty(t, lf.State.ActiveProfile)
}

func TestProfileApply_AlwaysInstalledKeepsBootstrapPackage(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()
	seedPackage(t, d, repo, "personal", "dots")
	seedPackage(t, d, repo, "personal", "zsh")
	_, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/dots"})
	require.NoError(t, err)

	writeConfig(t, d, "core:\n  always_installed: [personal/dots]\n")
	writeProfile(t, d, "name: base\npackages: [personal/dots]\n")
	// work omits and even excludes personal/dots; always_installed wins.
	writeProfile(t, d, "name: work\nextends: base\npackages: [personal/zsh]\nexclude: [personal/dots]\n")

	result, err := d.ProfileApply(ctx, dotsctl.ProfileApplyOptions{Name: "work"})
	require.NoError(t, err)
	require.Empty(t, result.Remove)
	require.Equal(t, []string{"personal/dots"}, result.Keep)
	require.Equal(t, []string{"personal/zsh"}, result.Install)
	require.ElementsMatch(t, []string{"personal/dots", "personal/zsh"}, installedRefs(t, repo))
}

func TestProfileApply_MissingProfile(t *testing.T) {
	d, _ := newTestDots(t)
	_, err := d.ProfileApply(context.Background(), dotsctl.ProfileApplyOptions{Name: "nope"})
	require.ErrorIs(t, err, dots.ErrNotExist)
}

func TestStatus_ActiveProfileFromLockfile(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()
	seedPackage(t, d, repo, "t", "a")
	writeProfile(t, d, "name: p\npackages: [t/a]\n")
	_, err := d.ProfileApply(ctx, dotsctl.ProfileApplyOptions{Name: "p"})
	require.NoError(t, err)

	status, err := d.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, "p", status.Profile)
}

// --- Init --apply ---

// seedBootstrapPackage creates a "dots" package in tap whose only link is the
// given config.yaml, mirroring the docs' dots package pattern.
func seedBootstrapPackage(t *testing.T, d *dotsctl.Dots, repo *dots.MemoryRepo, tap, config string) {
	t.Helper()
	pkgDir := filepath.Join(d.PathService.TapsDir(), tap, "dots")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	manifest := []byte("package:\n  name: dots\nlinks:\n  config.yaml: \"@config/dots/config.yaml\"\n")
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "Dotfile.yaml"), manifest, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "config.yaml"), []byte(config), 0o644))
	require.NoError(t, repo.AddTap(context.Background(), dots.TapInfo{Name: tap, URL: "test"}))
	require.NoError(t, repo.AddPackage(tap, dots.PackageInfo{Tap: tap, Name: "dots", Dir: "dots"}, manifest))
}

func TestInit_ApplyKeepsBootstrapPackage(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()
	seedBootstrapPackage(t, d, repo, "personal", "core:\n  always_installed: [personal/dots]\n")
	seedPackage(t, d, repo, "personal", "zsh")
	// work never mentions personal/dots; always_installed from the
	// bootstrapped config must keep it.
	writeProfile(t, d, "name: work\npackages: [personal/zsh]\n")

	err := d.Init(ctx, dotsctl.InitOptions{
		From:  "git@github.com:me/dotfiles.git",
		Name:  "personal",
		Path:  "dots",
		Apply: "work",
	})
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"personal/dots", "personal/zsh"}, installedRefs(t, repo))
	lf, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	require.Equal(t, "work", lf.State.ActiveProfile)
}

func TestInit_ApplyMissingProfileAfterBootstrap(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()
	seedBootstrapPackage(t, d, repo, "personal", "core: {}\n")

	err := d.Init(ctx, dotsctl.InitOptions{
		From:  "git@github.com:me/dotfiles.git",
		Name:  "personal",
		Path:  "dots",
		Apply: "nope",
	})
	require.ErrorIs(t, err, dots.ErrNotExist)
	require.Contains(t, err.Error(), "init completed")
	// The bootstrap install still happened.
	require.Equal(t, []string{"personal/dots"}, installedRefs(t, repo))
}
