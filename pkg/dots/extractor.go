package dots

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/ulikunitz/xz"
)

// ExtractRequest is the unit of work passed to Extract: the raw bytes of an
// artifact (verified by the fetcher), the archive shape, and the destination
// directory. StripComponents drops N leading path elements from every entry,
// matching tar(1) semantics — entries with fewer than N elements are skipped.
type ExtractRequest struct {
	Format          ExtractFormat
	StripComponents int
	// Executables marks staged paths (after StripComponents) that should be
	// chmod'd 0755 on Unix. ExtractNone treats Stage as the destination
	// filename and ignores Executables; the artifact-level executable bit
	// is set via the manifest's enclosing entry.
	Executables map[string]struct{}
	Bytes       []byte
	// Stage is the on-disk filename for ExtractNone. Ignored for archive
	// formats which dictate their own internal layout.
	Stage string
	// DestDir is the absolute directory the extractor writes into.
	DestDir string
}

// Extract dispatches by Format. The destination directory is created if it
// does not exist; existing files inside are overwritten without prompting,
// which matches the lockfile-driven reinstall flow. Filesystem I/O routes
// through the runtime so jailed sandboxes capture writes.
func Extract(rt *toolkit.Runtime, req ExtractRequest) error {
	if rt == nil {
		return fmt.Errorf("%w: extractor runtime is required", ErrInvalid)
	}
	if err := rt.Mkdir(req.DestDir, 0o755, true); err != nil {
		return fmt.Errorf("artifact dest %s: %w", req.DestDir, err)
	}
	switch req.Format {
	case ExtractNone:
		return extractNone(rt, req)
	case ExtractTarGz:
		gz, err := gzip.NewReader(bytes.NewReader(req.Bytes))
		if err != nil {
			return fmt.Errorf("gzip reader: %w", err)
		}
		defer gz.Close()
		return extractTar(rt, tar.NewReader(gz), req)
	case ExtractTarXz:
		xr, err := xz.NewReader(bytes.NewReader(req.Bytes))
		if err != nil {
			return fmt.Errorf("xz reader: %w", err)
		}
		return extractTar(rt, tar.NewReader(xr), req)
	case ExtractZip:
		return extractZip(rt, req)
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedExtract, req.Format)
	}
}

// extractNone writes the raw bytes to DestDir/Stage. Stage is required
// here because there is no archive layout to fall back on; the loader is
// expected to default Stage to filepath.Base(URL) when the manifest leaves
// it unset.
func extractNone(rt *toolkit.Runtime, req ExtractRequest) error {
	if req.Stage == "" {
		return fmt.Errorf("%w: extract=none requires stage", ErrInvalid)
	}
	dest := filepath.Join(req.DestDir, req.Stage)
	if err := rt.Mkdir(filepath.Dir(dest), 0o755, true); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if _, ok := req.Executables[req.Stage]; ok {
		mode = 0o755
	}
	return rt.WriteFile(dest, req.Bytes, mode)
}

// extractTar walks tar headers, applies StripComponents, and writes regular
// files plus directories. Symlinks and other special types are skipped
// silently — release tarballs that include them generally also include the
// pointed-to file as a regular entry, which is what we want anyway.
func extractTar(rt *toolkit.Runtime, tr *tar.Reader, req ExtractRequest) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("tar next: %w", err)
		}
		name, ok := stripComponents(hdr.Name, req.StripComponents)
		if !ok {
			continue
		}
		dest, err := safeJoin(req.DestDir, name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := rt.Mkdir(dest, 0o755, true); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := rt.Mkdir(filepath.Dir(dest), 0o755, true); err != nil {
				return err
			}
			f, err := rt.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode(name, hdr.Mode, req.Executables))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		default:
			// Skip symlinks, hardlinks, devices.
		}
	}
}

// extractZip mirrors extractTar's semantics: regular files and directories
// only, StripComponents applied, executable bit honored.
func extractZip(rt *toolkit.Runtime, req ExtractRequest) error {
	zr, err := zip.NewReader(bytes.NewReader(req.Bytes), int64(len(req.Bytes)))
	if err != nil {
		return fmt.Errorf("zip reader: %w", err)
	}
	for _, entry := range zr.File {
		name, ok := stripComponents(entry.Name, req.StripComponents)
		if !ok {
			continue
		}
		dest, err := safeJoin(req.DestDir, name)
		if err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			if err := rt.Mkdir(dest, 0o755, true); err != nil {
				return err
			}
			continue
		}
		if err := rt.Mkdir(filepath.Dir(dest), 0o755, true); err != nil {
			return err
		}
		rc, err := entry.Open()
		if err != nil {
			return err
		}
		f, err := rt.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode(name, int64(entry.Mode().Perm()), req.Executables))
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(f, rc); err != nil {
			rc.Close()
			f.Close()
			return err
		}
		rc.Close()
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

// stripComponents drops N leading path elements. Entries that don't have
// enough components are not an error; they're silently skipped, which
// matches GNU tar's `--strip-components` behavior.
func stripComponents(name string, n int) (string, bool) {
	if n <= 0 {
		return name, name != ""
	}
	parts := strings.Split(strings.TrimPrefix(name, "/"), "/")
	if len(parts) <= n {
		return "", false
	}
	rest := strings.Join(parts[n:], "/")
	if rest == "" {
		return "", false
	}
	return rest, true
}

// safeJoin refuses paths that would escape the destination via "..", and
// rejects absolute paths outright. This is the canonical zip-slip /
// tar-slip mitigation: an attacker-controlled archive entry cannot write
// outside DestDir, even if it has been path-cleaned by the standard
// library. We check the raw entry name BEFORE cleaning because Clean
// collapses "../../etc/passwd" into "/etc/passwd", which would then
// (mis)pass a base-prefix check.
func safeJoin(base, rel string) (string, error) {
	if strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) {
		return "", fmt.Errorf("%w: archive entry has absolute path: %s", ErrInvalid, rel)
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == ".." {
			return "", fmt.Errorf("%w: archive entry escapes dest: %s", ErrInvalid, rel)
		}
	}
	joined := filepath.Join(base, filepath.FromSlash(rel))
	relPath, err := filepath.Rel(base, joined)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("%w: archive entry escapes dest: %s", ErrInvalid, rel)
	}
	return joined, nil
}

// fileMode chooses the on-disk mode for an extracted entry. Manifest-marked
// executables override the archive's recorded bit so a release published
// without the exec bit (a common Windows-uploader bug) still ends up
// runnable on Unix.
func fileMode(name string, archiveMode int64, execs map[string]struct{}) os.FileMode {
	if _, ok := execs[name]; ok {
		return 0o755
	}
	if archiveMode == 0 {
		return 0o644
	}
	mode := os.FileMode(archiveMode).Perm()
	if mode == 0 {
		return 0o644
	}
	return mode
}
