package dotsctl_test

import (
	"context"
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/stretchr/testify/require"
)

func TestTapRemove_UninstallsPackages(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedPackage(t, d, repo, "personal", "nvim")
	seedPackage(t, d, repo, "personal", "git")
	seedPackage(t, d, repo, "work", "ssh")

	_, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/nvim"})
	require.NoError(t, err)
	_, err = d.Install(ctx, dotsctl.InstallOptions{Package: "personal/git"})
	require.NoError(t, err)
	_, err = d.Install(ctx, dotsctl.InstallOptions{Package: "work/ssh"})
	require.NoError(t, err)

	result, err := d.TapRemove(ctx, "personal")
	require.NoError(t, err)
	require.True(t, result.TapExisted)
	require.ElementsMatch(t, []string{"personal/nvim", "personal/git"}, result.Uninstalled)
	require.Empty(t, result.Failed)

	// Only the unrelated work/ssh package should remain in the lockfile.
	lockfile, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	require.Len(t, lockfile.Installed, 1)
	require.Equal(t, "work/ssh", lockfile.Installed[0].Package)

	// The tap itself is gone.
	_, err = repo.GetTap(ctx, "personal")
	require.Error(t, err)
}

func TestTapRemove_OrphanLockfile(t *testing.T) {
	// Simulate the user's recovery case: the on-disk tap was already removed
	// (e.g. by a prior `dots tap remove` that didn't cascade) but the lockfile
	// still carries an install entry. `TapRemove` must clean it up.
	d, repo := newTestDots(t)
	ctx := context.Background()

	seedPackage(t, d, repo, "default", "dots-config")
	_, err := d.Install(ctx, dotsctl.InstallOptions{Package: "default/dots-config"})
	require.NoError(t, err)

	// Drop the tap behind TapRemove's back.
	require.NoError(t, repo.RemoveTap(ctx, "default"))

	result, err := d.TapRemove(ctx, "default")
	require.NoError(t, err)
	require.False(t, result.TapExisted)
	require.Equal(t, []string{"default/dots-config"}, result.Uninstalled)

	lockfile, err := repo.ReadLockfile(ctx)
	require.NoError(t, err)
	require.Empty(t, lockfile.Installed)
}

func TestTapRemove_NoLockfile(t *testing.T) {
	// No installs, no lockfile — TapRemove should still drop the tap.
	d, repo := newTestDots(t)
	ctx := context.Background()

	require.NoError(t, repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"}))

	result, err := d.TapRemove(ctx, "personal")
	require.NoError(t, err)
	require.True(t, result.TapExisted)
	require.Empty(t, result.Uninstalled)
	require.Empty(t, result.Failed)
}

func TestTapRemove_EmptyName(t *testing.T) {
	d, _ := newTestDots(t)
	_, err := d.TapRemove(context.Background(), "")
	require.Error(t, err)
}
