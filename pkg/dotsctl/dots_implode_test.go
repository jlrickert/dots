package dotsctl_test

import (
	"context"
	"testing"

	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/stretchr/testify/require"
)

func TestImplode_RequiresYes(t *testing.T) {
	d, _ := newTestDots(t)
	_, err := d.Implode(context.Background(), dotsctl.ImplodeOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "--yes")
}

func TestImplode_UninstallsPackagesAndClearsState(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	require.NoError(t, d.Init(ctx, dotsctl.InitOptions{}))

	seedPackage(t, d, repo, "personal", "nvim")
	seedPackage(t, d, repo, "personal", "git")
	seedPackage(t, d, repo, "work", "ssh")

	_, err := d.Install(ctx, dotsctl.InstallOptions{Package: "personal/nvim"})
	require.NoError(t, err)
	_, err = d.Install(ctx, dotsctl.InstallOptions{Package: "personal/git"})
	require.NoError(t, err)
	_, err = d.Install(ctx, dotsctl.InstallOptions{Package: "work/ssh"})
	require.NoError(t, err)

	result, err := d.Implode(ctx, dotsctl.ImplodeOptions{Yes: true})
	require.NoError(t, err)
	require.ElementsMatch(
		t,
		[]string{"personal/nvim", "personal/git", "work/ssh"},
		result.Uninstalled,
	)
	require.Empty(t, result.Failed)
	require.True(t, result.StateDirRemoved)
	require.True(t, result.ConfigDirRemoved)

	// Second implode on the same dots instance is a no-op.
	result2, err := d.Implode(ctx, dotsctl.ImplodeOptions{Yes: true})
	require.NoError(t, err)
	require.Empty(t, result2.Uninstalled)
	require.False(t, result2.StateDirRemoved)
	require.False(t, result2.ConfigDirRemoved)
}

func TestImplode_NoStateYet(t *testing.T) {
	d, _ := newTestDots(t)
	ctx := context.Background()

	result, err := d.Implode(ctx, dotsctl.ImplodeOptions{Yes: true})
	require.NoError(t, err)
	require.Empty(t, result.Uninstalled)
	require.Empty(t, result.Failed)
	require.False(t, result.StateDirRemoved)
	require.False(t, result.ConfigDirRemoved)
}

func TestImplode_InitializedButEmpty(t *testing.T) {
	d, _ := newTestDots(t)
	ctx := context.Background()

	require.NoError(t, d.Init(ctx, dotsctl.InitOptions{}))

	result, err := d.Implode(ctx, dotsctl.ImplodeOptions{Yes: true})
	require.NoError(t, err)
	require.Empty(t, result.Uninstalled)
	require.True(t, result.StateDirRemoved)
	require.True(t, result.ConfigDirRemoved)
}
