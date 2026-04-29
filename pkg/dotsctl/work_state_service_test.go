package dotsctl_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/dots/pkg/dots"
	"github.com/jlrickert/dots/pkg/dotsctl"
	"github.com/stretchr/testify/require"
)

// testWorkStateFixture wires a WorkStateService against a jailed test runtime
// and exposes both the virtual path used by the service and the host path
// where the jail actually lays the file down on disk.
type testWorkStateFixture struct {
	svc      *dotsctl.WorkStateService
	rt       *toolkit.Runtime
	jail     string
	virtual  string
	hostPath string
}

func newTestWorkStateService(t *testing.T) *testWorkStateFixture {
	t.Helper()
	dir := t.TempDir()
	rt, err := toolkit.NewTestRuntime(dir, dir, "testuser")
	require.NoError(t, err)

	platform := dots.DetectPlatform()
	ps := dotsctl.NewPathService(platform, rt)
	virtual := ps.WorkStateFile()
	svc := dotsctl.NewWorkStateService(ps, virtual, rt)
	return &testWorkStateFixture{
		svc:      svc,
		rt:       rt,
		jail:     dir,
		virtual:  virtual,
		hostPath: filepath.Join(dir, virtual),
	}
}

func TestWorkStateService_LoadMissingReturnsDefault(t *testing.T) {
	f := newTestWorkStateService(t)
	state, err := f.svc.Load(false)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, state.Taps)
	require.Empty(t, state.Taps)
}

func TestWorkStateService_SaveLoadRoundtrip(t *testing.T) {
	f := newTestWorkStateService(t)

	in := &dots.WorkState{Taps: map[string]string{
		"personal": "/home/me/dots",
		"work":     "/Users/me/work-dots",
	}}
	require.NoError(t, f.svc.Save(in))

	f.svc.InvalidateCache()
	out, err := f.svc.Load(false)
	require.NoError(t, err)
	require.Equal(t, "/home/me/dots", out.Taps["personal"])
	require.Equal(t, "/Users/me/work-dots", out.Taps["work"])
}

func TestWorkStateService_SaveWritesModeline(t *testing.T) {
	f := newTestWorkStateService(t)

	require.NoError(t, f.svc.Save(&dots.WorkState{Taps: map[string]string{"a": "/tmp/a"}}))

	raw, err := os.ReadFile(f.hostPath)
	require.NoError(t, err)
	require.True(t,
		strings.HasPrefix(string(raw), dots.DotsWorkStateSchemaModeline),
		"saved file must start with the schema modeline; got %q", string(raw))
}

func TestWorkStateService_SaveCreatesParentDir(t *testing.T) {
	f := newTestWorkStateService(t)

	// Parent dir should not exist yet — Save must create it via AtomicWriteFile.
	parent := filepath.Dir(f.hostPath)
	_, err := os.Stat(parent)
	require.True(t, os.IsNotExist(err), "parent dir should not exist before Save: %v", err)

	require.NoError(t, f.svc.Save(&dots.WorkState{Taps: map[string]string{"a": "/tmp/a"}}))

	info, err := os.Stat(parent)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestWorkStateService_SetGetDeleteAll(t *testing.T) {
	f := newTestWorkStateService(t)

	require.NoError(t, f.svc.Set("personal", "/home/me/dots"))
	require.NoError(t, f.svc.Set("work", "/Users/me/work-dots"))

	got, ok := f.svc.Get("personal")
	require.True(t, ok)
	require.Equal(t, "/home/me/dots", got)

	all, err := f.svc.All()
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, "/home/me/dots", all["personal"])

	// Mutating returned map must not affect service state.
	all["mutated"] = "/oops"
	got2, ok := f.svc.Get("mutated")
	require.False(t, ok)
	require.Equal(t, "", got2)

	require.NoError(t, f.svc.Delete("personal"))
	_, ok = f.svc.Get("personal")
	require.False(t, ok)

	all, err = f.svc.All()
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Contains(t, all, "work")
}

func TestWorkStateService_GetMissing(t *testing.T) {
	f := newTestWorkStateService(t)
	_, ok := f.svc.Get("nope")
	require.False(t, ok)
}
