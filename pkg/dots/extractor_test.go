package dots_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

// newExtractorRuntime builds an unjailed Runtime suitable for extractor
// tests. The extractor writes to absolute t.TempDir() paths, so the
// runtime's filesystem seam needs to pass them through unchanged — which is
// exactly what NewOsRuntime gives us.
func newExtractorRuntime(t *testing.T) *toolkit.Runtime {
	t.Helper()
	rt, err := toolkit.NewOsRuntime()
	require.NoError(t, err)
	return rt
}

// tarEntry is a small spec used by buildTar to drive table-driven cases.
type tarEntry struct {
	name string
	body string
	mode int64
	dir  bool
}

func buildTar(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: e.mode}
		if e.dir {
			hdr.Typeflag = tar.TypeDir
			hdr.Name = e.name
		} else {
			hdr.Typeflag = tar.TypeReg
			hdr.Size = int64(len(e.body))
		}
		require.NoError(t, tw.WriteHeader(hdr))
		if !e.dir {
			_, err := tw.Write([]byte(e.body))
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	return buf.Bytes()
}

func gzipBytes(t *testing.T, raw []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write(raw)
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

func xzBytes(t *testing.T, raw []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	xw, err := xz.NewWriter(&buf)
	require.NoError(t, err)
	_, err = xw.Write(raw)
	require.NoError(t, err)
	require.NoError(t, xw.Close())
	return buf.Bytes()
}

func buildZip(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		if e.dir {
			_, err := zw.Create(e.name + "/")
			require.NoError(t, err)
			continue
		}
		hdr := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if e.mode != 0 {
			hdr.SetMode(os.FileMode(e.mode).Perm())
		}
		w, err := zw.CreateHeader(hdr)
		require.NoError(t, err)
		_, err = w.Write([]byte(e.body))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestExtractor_TarGz(t *testing.T) {
	entries := []tarEntry{
		{name: "release/", dir: true},
		{name: "release/README", body: "readme", mode: 0o644},
		{name: "release/bin/dots", body: "binary-bytes", mode: 0o755},
	}
	dest := t.TempDir()
	require.NoError(t, dots.Extract(newExtractorRuntime(t), dots.ExtractRequest{
		Format:  dots.ExtractTarGz,
		Bytes:   gzipBytes(t, buildTar(t, entries)),
		DestDir: dest,
	}))

	readme, err := os.ReadFile(filepath.Join(dest, "release", "README"))
	require.NoError(t, err)
	require.Equal(t, "readme", string(readme))
}

func TestExtractor_TarXz(t *testing.T) {
	entries := []tarEntry{
		{name: "tool", body: "xz-payload", mode: 0o644},
	}
	dest := t.TempDir()
	require.NoError(t, dots.Extract(newExtractorRuntime(t), dots.ExtractRequest{
		Format:  dots.ExtractTarXz,
		Bytes:   xzBytes(t, buildTar(t, entries)),
		DestDir: dest,
	}))

	got, err := os.ReadFile(filepath.Join(dest, "tool"))
	require.NoError(t, err)
	require.Equal(t, "xz-payload", string(got))
}

func TestExtractor_Zip(t *testing.T) {
	entries := []tarEntry{
		{name: "pkg/data.txt", body: "zipped"},
	}
	dest := t.TempDir()
	require.NoError(t, dots.Extract(newExtractorRuntime(t), dots.ExtractRequest{
		Format:  dots.ExtractZip,
		Bytes:   buildZip(t, entries),
		DestDir: dest,
	}))

	got, err := os.ReadFile(filepath.Join(dest, "pkg", "data.txt"))
	require.NoError(t, err)
	require.Equal(t, "zipped", string(got))
}

func TestExtractor_None(t *testing.T) {
	dest := t.TempDir()
	require.NoError(t, dots.Extract(newExtractorRuntime(t), dots.ExtractRequest{
		Format:  dots.ExtractNone,
		Bytes:   []byte("raw-binary"),
		Stage:   "dots",
		DestDir: dest,
	}))

	got, err := os.ReadFile(filepath.Join(dest, "dots"))
	require.NoError(t, err)
	require.Equal(t, "raw-binary", string(got))
}

func TestExtractor_StripComponents(t *testing.T) {
	entries := []tarEntry{
		{name: "release-v1.0.0/", dir: true},
		{name: "release-v1.0.0/bin/", dir: true},
		{name: "release-v1.0.0/bin/dots", body: "x", mode: 0o755},
		{name: "release-v1.0.0/README", body: "r", mode: 0o644},
	}
	dest := t.TempDir()
	require.NoError(t, dots.Extract(newExtractorRuntime(t), dots.ExtractRequest{
		Format:          dots.ExtractTarGz,
		StripComponents: 1,
		Bytes:           gzipBytes(t, buildTar(t, entries)),
		DestDir:         dest,
	}))

	// Top-level "release-v1.0.0" prefix is gone.
	_, err := os.Stat(filepath.Join(dest, "bin", "dots"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dest, "README"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dest, "release-v1.0.0"))
	require.True(t, errors.Is(err, os.ErrNotExist))
}

func TestExtractor_ExecutableBit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit not modeled on windows")
	}
	entries := []tarEntry{
		{name: "bin/dots", body: "x", mode: 0o644},
	}
	dest := t.TempDir()
	require.NoError(t, dots.Extract(newExtractorRuntime(t), dots.ExtractRequest{
		Format: dots.ExtractTarGz,
		Bytes:  gzipBytes(t, buildTar(t, entries)),
		Executables: map[string]struct{}{
			"bin/dots": {},
		},
		DestDir: dest,
	}))

	info, err := os.Stat(filepath.Join(dest, "bin", "dots"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), info.Mode().Perm(),
		"manifest-marked executable should override archive-recorded 0644")
}

func TestExtractor_Unsupported(t *testing.T) {
	err := dots.Extract(newExtractorRuntime(t), dots.ExtractRequest{
		Format:  "rar",
		Bytes:   []byte{},
		DestDir: t.TempDir(),
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, dots.ErrUnsupportedExtract))
}

func TestExtractor_NoneRequiresStage(t *testing.T) {
	err := dots.Extract(newExtractorRuntime(t), dots.ExtractRequest{
		Format:  dots.ExtractNone,
		Bytes:   []byte("x"),
		DestDir: t.TempDir(),
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, dots.ErrInvalid))
}

func TestExtractor_TarSlipBlocked(t *testing.T) {
	// An attacker tarball that tries to write outside DestDir.
	entries := []tarEntry{
		{name: "../../etc/passwd", body: "pwned", mode: 0o644},
	}
	dest := t.TempDir()
	err := dots.Extract(newExtractorRuntime(t), dots.ExtractRequest{
		Format:  dots.ExtractTarGz,
		Bytes:   gzipBytes(t, buildTar(t, entries)),
		DestDir: dest,
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, dots.ErrInvalid))
}
