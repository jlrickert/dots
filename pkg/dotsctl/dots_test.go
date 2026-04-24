package dotsctl_test

import (
	"context"
	"testing"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/dots/pkg/dots"
	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/stretchr/testify/require"
)

func newTestDots(t *testing.T) (*dotsctl.Dots, *dots.MemoryRepo) {
	t.Helper()
	dir := t.TempDir()
	rt, err := toolkit.NewTestRuntime(dir, dir, "testuser")
	require.NoError(t, err)

	repo := dots.NewMemoryRepo()
	d, err := dotsctl.NewDots(dotsctl.DotsOptions{
		Runtime: rt,
		Repo:    repo,
	})
	require.NoError(t, err)
	return d, repo
}

func TestNewDots(t *testing.T) {
	d, _ := newTestDots(t)
	require.NotNil(t, d)
	require.NotNil(t, d.PathService)
	require.NotNil(t, d.ConfigService)
	require.NotNil(t, d.Repo)
}

func TestNewDots_NilRuntime(t *testing.T) {
	_, err := dotsctl.NewDots(dotsctl.DotsOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "runtime is required")
}

// --- List ---

func TestList_EmptyInstalled(t *testing.T) {
	d, _ := newTestDots(t)
	ctx := context.Background()

	result, err := d.List(ctx, dotsctl.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, result.Installed)
}

func TestList_InstalledPackages(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	err := repo.WriteLockfile(ctx, &dots.Lockfile{
		Installed: []dots.InstalledPackage{
			{Package: "personal/nvim", Tap: "personal"},
			{Package: "work/ssh", Tap: "work"},
		},
	})
	require.NoError(t, err)

	result, err := d.List(ctx, dotsctl.ListOptions{})
	require.NoError(t, err)
	require.Len(t, result.Installed, 2)
}

func TestList_FilterByTap(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	err := repo.WriteLockfile(ctx, &dots.Lockfile{
		Installed: []dots.InstalledPackage{
			{Package: "personal/nvim", Tap: "personal"},
			{Package: "work/ssh", Tap: "work"},
			{Package: "personal/git", Tap: "personal"},
		},
	})
	require.NoError(t, err)

	result, err := d.List(ctx, dotsctl.ListOptions{Tap: "personal"})
	require.NoError(t, err)
	require.Len(t, result.Installed, 2)
}

func TestList_Available(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)
	err = repo.AddPackage("personal", dots.PackageInfo{Tap: "personal", Name: "nvim"}, nil)
	require.NoError(t, err)
	err = repo.AddPackage("personal", dots.PackageInfo{Tap: "personal", Name: "git"}, nil)
	require.NoError(t, err)

	result, err := d.List(ctx, dotsctl.ListOptions{Available: true})
	require.NoError(t, err)
	require.Len(t, result.Available, 2)
}

func TestList_AvailableByTap(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)
	err = repo.AddTap(ctx, dots.TapInfo{Name: "work", URL: "url2"})
	require.NoError(t, err)
	err = repo.AddPackage("personal", dots.PackageInfo{Tap: "personal", Name: "nvim"}, nil)
	require.NoError(t, err)
	err = repo.AddPackage("work", dots.PackageInfo{Tap: "work", Name: "ssh"}, nil)
	require.NoError(t, err)

	result, err := d.List(ctx, dotsctl.ListOptions{Available: true, Tap: "work"})
	require.NoError(t, err)
	require.Len(t, result.Available, 1)
	require.Equal(t, "ssh", result.Available[0].Name)
}

// --- Status ---

func TestStatus(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	err := repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"})
	require.NoError(t, err)

	result, err := d.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, result.TapCount)
	require.Equal(t, 0, result.PackageCount)
	require.Equal(t, dots.LinkCopy, result.LinkStrategy)
}

// --- Init ---

func TestInit_CreatesDirectories(t *testing.T) {
	d, _ := newTestDots(t)
	ctx := context.Background()

	err := d.Init(ctx, dotsctl.InitOptions{})
	require.NoError(t, err)
}

func TestInit_WithFrom(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	err := d.Init(ctx, dotsctl.InitOptions{
		From: "git@github.com:me/dotfiles.git",
	})
	require.NoError(t, err)

	tap, err := repo.GetTap(ctx, "default")
	require.NoError(t, err)
	require.Equal(t, "git@github.com:me/dotfiles.git", tap.URL)
}

func TestInit_WithFromAndName(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	err := d.Init(ctx, dotsctl.InitOptions{
		From: "git@github.com:jlrickert/dot-personal.git",
		Name: "private",
	})
	require.NoError(t, err)

	tap, err := repo.GetTap(ctx, "private")
	require.NoError(t, err)
	require.Equal(t, "git@github.com:jlrickert/dot-personal.git", tap.URL)

	_, err = repo.GetTap(ctx, "default")
	require.Error(t, err, "default tap should not be created when --name is set")
}

// --- Doctor ---

func TestDoctor(t *testing.T) {
	d, _ := newTestDots(t)
	ctx := context.Background()

	checks, err := d.Doctor(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, checks)

	// Should have at least config dir, state dir, config file, and taps checks.
	require.GreaterOrEqual(t, len(checks), 4)
}
