package dotsctl_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/dots/pkg/dots"
	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/stretchr/testify/require"
)

func TestWorkOn_ExpandsTildeInLocalPath(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	require.NoError(t, repo.AddTap(ctx, dots.TapInfo{Name: "personal", URL: "url"}))

	err := d.WorkOn(ctx, dotsctl.WorkOnOptions{
		Tap:       "personal",
		LocalPath: "~/repos/my-tap",
	})
	require.NoError(t, err)

	home, err := d.Runtime.GetHome()
	require.NoError(t, err)

	path, ok := d.WorkStateService.Get("personal")
	require.True(t, ok)
	require.Equal(t, filepath.Join(home, "repos/my-tap"), path)
}

func TestWorkOn_MigratesLegacyConfigWorkMode(t *testing.T) {
	d, repo := newTestDots(t)
	ctx := context.Background()

	require.NoError(t, repo.AddTap(ctx, dots.TapInfo{Name: "legacy", URL: "url-legacy"}))
	require.NoError(t, repo.AddTap(ctx, dots.TapInfo{Name: "fresh", URL: "url-fresh"}))

	// Seed config.yaml with a legacy work_mode entry to simulate a config that
	// pre-dates the work-state file split.
	cfg, err := d.ConfigService.Config(false)
	require.NoError(t, err)
	cfg.WorkMode = map[string]string{"legacy": "/tmp/legacy-checkout"}
	require.NoError(t, d.ConfigService.Save(cfg))

	// A WorkOn call for a different tap should still trigger migration.
	require.NoError(t, d.WorkOn(ctx, dotsctl.WorkOnOptions{
		Tap:       "fresh",
		LocalPath: "/tmp/fresh-checkout",
	}))

	legacyPath, ok := d.WorkStateService.Get("legacy")
	require.True(t, ok, "legacy work_mode entry should be migrated to state")
	require.Equal(t, "/tmp/legacy-checkout", legacyPath)

	freshPath, ok := d.WorkStateService.Get("fresh")
	require.True(t, ok, "new work-mode entry should be in state")
	require.Equal(t, "/tmp/fresh-checkout", freshPath)

	d.ConfigService.InvalidateCache()
	cfg, err = d.ConfigService.Config(false)
	require.NoError(t, err)
	require.Empty(t, cfg.WorkMode, "config.yaml work_mode must be cleared after migration")
}

func TestNewDots_ExpandsTildeInConfigPath(t *testing.T) {
	dir := t.TempDir()
	rt, err := toolkit.NewTestRuntime(dir, dir, "testuser")
	require.NoError(t, err)

	d, err := dotsctl.NewDots(dotsctl.DotsOptions{
		Runtime:    rt,
		ConfigPath: "~/custom/config.yaml",
		Repo:       dots.NewMemoryRepo(),
	})
	require.NoError(t, err)

	home, err := rt.GetHome()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, "custom/config.yaml"), d.ConfigService.ConfigPath)
}
