package dots

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LinkAction describes a single link operation to perform.
type LinkAction struct {
	Src      string       // absolute source path (file in tap package)
	Dest     string       // absolute target path (resolved alias path)
	Strategy LinkStrategy // symlink, copy, or hardlink
	Origin   string       // platform cascade origin (base, darwin, etc.)
}

// LinkResult records the outcome of a link action.
type LinkResult struct {
	Src      string
	Dest     string
	Method   string
	Checksum string
	Origin   string
}

// ResolveLinkActions resolves manifest links to concrete source/dest paths.
// pkgDir is the absolute path to the package directory within the tap.
// resolver expands alias paths to absolute paths.
// strategy is the effective link strategy for this package.
func ResolveLinkActions(
	resolved *ResolvedManifest,
	pkgDir string,
	resolver *AliasResolver,
	strategy LinkStrategy,
) ([]LinkAction, error) {
	var actions []LinkAction
	for src, dest := range resolved.Links {
		absSrc := filepath.Join(pkgDir, filepath.FromSlash(src))

		var absDest string
		if strings.HasPrefix(dest, "@") {
			var err error
			absDest, err = resolver.Resolve(dest)
			if err != nil {
				return nil, fmt.Errorf("resolve alias %q: %w", dest, err)
			}
		} else {
			// Relative paths are relative to home
			var err error
			absDest, err = resolver.Resolve(dest)
			if err != nil {
				return nil, fmt.Errorf("resolve path %q: %w", dest, err)
			}
		}

		actions = append(actions, LinkAction{
			Src:      absSrc,
			Dest:     absDest,
			Strategy: strategy,
		})
	}
	return actions, nil
}

// PlaceLink executes a single link action, creating the target file.
// It creates parent directories as needed.
func PlaceLink(action LinkAction) (*LinkResult, error) {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(action.Dest), 0o755); err != nil {
		return nil, fmt.Errorf("create parent dir for %s: %w", action.Dest, err)
	}

	var method string
	var checksum string

	switch action.Strategy {
	case LinkSymlink:
		if err := os.Symlink(action.Src, action.Dest); err != nil {
			return nil, fmt.Errorf("symlink %s -> %s: %w", action.Dest, action.Src, err)
		}
		method = "symlink"
	case LinkCopy:
		cs, err := copyFile(action.Src, action.Dest)
		if err != nil {
			return nil, fmt.Errorf("copy %s -> %s: %w", action.Src, action.Dest, err)
		}
		method = "copy"
		checksum = cs
	case LinkHardlink:
		if err := os.Link(action.Src, action.Dest); err != nil {
			return nil, fmt.Errorf("hardlink %s -> %s: %w", action.Dest, action.Src, err)
		}
		method = "hardlink"
	default:
		return nil, fmt.Errorf("unknown link strategy: %s", action.Strategy)
	}

	return &LinkResult{
		Src:      action.Src,
		Dest:     action.Dest,
		Method:   method,
		Checksum: checksum,
		Origin:   action.Origin,
	}, nil
}

// RemoveLink removes a placed link/copy at the given path.
func RemoveLink(path string) error {
	return os.Remove(path)
}

// ReadFileForBackup reads a file's contents for backup purposes.
func ReadFileForBackup(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// RestoreFileFromBackup writes backup data back to the original path.
func RestoreFileFromBackup(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func copyFile(src, dest string) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return "", err
	}

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return "", err
	}
	defer out.Close()

	h := sha256.New()
	w := io.MultiWriter(out, h)
	if _, err := io.Copy(w, in); err != nil {
		return "", err
	}

	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

// FileChecksum computes the sha256 checksum of a file.
func FileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
