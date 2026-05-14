package dots

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// LinkAction describes a single link operation to perform.
//
// File sources collapse to the historical (Strategy-driven) flow regardless
// of Mode. Directory sources route on Mode: a directory with mode=symlink
// creates one symlink at the directory root; a directory with mode=copy
// walks the tree and emits one InstalledFile per regular leaf so
// remove/diff/which keep working unchanged.
type LinkAction struct {
	Src      string       // absolute source path (file or dir in tap package)
	Dest     string       // absolute target path (resolved alias path)
	Strategy LinkStrategy // symlink, copy, or hardlink — drives single-file flow
	Origin   string       // platform cascade origin (base, darwin, etc.)

	// Mode is the directory-link discriminator from the manifest. Empty
	// (LinkModeAuto) is resolved against Strategy at link time when IsDir
	// is true.
	Mode LinkMode
	// Exclude is the user-supplied glob list applied during a recursive
	// directory copy. Ignored when IsDir is false or Mode is symlink.
	Exclude []string
	// IsDir is set by ResolveLinkActions after stat'ing Src so the linker
	// does not need to re-stat. Symlinks-to-dirs count as directories.
	IsDir bool
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
//
// Each spec is stat'd here (not at PlaceLink time) so the directory-vs-file
// decision is fixed before any destructive action and so dry-run output can
// reflect what would actually happen. Missing sources are not an error
// here: the per-action error will surface once PlaceLink tries to use them,
// matching the previous file-only behavior.
func ResolveLinkActions(
	resolved *ResolvedManifest,
	pkgDir string,
	resolver *AliasResolver,
	strategy LinkStrategy,
) ([]LinkAction, error) {
	var actions []LinkAction
	for src, spec := range resolved.Links {
		absSrc := filepath.Join(pkgDir, filepath.FromSlash(src))

		// Both @alias and raw paths flow through resolver.Resolve so the
		// branch below is uniform; kept as a single call for clarity.
		absDest, err := resolver.Resolve(spec.Target)
		if err != nil {
			return nil, fmt.Errorf("resolve path %q: %w", spec.Target, err)
		}

		// Stat the source to lock in IsDir. os.Stat (not Lstat) follows
		// symlinks-to-dirs which behave as directories for our purposes.
		isDir := false
		if info, err := os.Stat(absSrc); err == nil {
			isDir = info.IsDir()
		}

		actions = append(actions, LinkAction{
			Src:      absSrc,
			Dest:     absDest,
			Strategy: strategy,
			Mode:     spec.Mode,
			Exclude:  spec.Exclude,
			IsDir:    isDir,
		})
	}
	return actions, nil
}

// PlaceLink executes a single link action, creating the target file(s).
// It creates parent directories as needed and returns one LinkResult per
// installed leaf:
//
//   - File source → one result, Method ∈ {"symlink", "copy", "hardlink"}.
//   - Directory source, mode=symlink → one result, Method="symlink-dir".
//   - Directory source, mode=copy → N results, one per regular file leaf,
//     Method="copy-dir-leaf" with sha256 checksum.
//
// LinkModeAuto is resolved here against the action's LinkStrategy:
// LinkSymlink → symlink-dir, LinkCopy → copy-dir, LinkHardlink → degrades
// to symlink-dir with a stderr warning (hardlinks can't span directories).
func PlaceLink(action LinkAction) ([]LinkResult, error) {
	if !action.IsDir {
		return placeFileLink(action)
	}

	// Resolve auto mode against the package's LinkStrategy. The hardlink
	// fallback is intentionally lossy: we don't error because a manifest
	// with link_strategy: hardlink and a directory entry is otherwise
	// useful, and downgrading to symlink matches the existing windows-
	// fallback ergonomics.
	mode := action.Mode
	if mode == LinkModeAuto {
		switch action.Strategy {
		case LinkSymlink:
			mode = LinkModeSymlink
		case LinkCopy:
			mode = LinkModeCopy
		case LinkHardlink:
			fmt.Fprintln(os.Stderr, "warning: hardlink strategy degraded to symlink for directory "+action.Src)
			mode = LinkModeSymlink
		default:
			return nil, fmt.Errorf("unknown link strategy: %s", action.Strategy)
		}
	}

	switch mode {
	case LinkModeSymlink:
		return placeDirSymlink(action)
	case LinkModeCopy:
		return placeDirCopy(action)
	default:
		return nil, fmt.Errorf("unknown link mode: %s", mode)
	}
}

// placeFileLink is the historical single-file flow, lifted out so PlaceLink
// can return []LinkResult uniformly.
func placeFileLink(action LinkAction) ([]LinkResult, error) {
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

	return []LinkResult{{
		Src:      action.Src,
		Dest:     action.Dest,
		Method:   method,
		Checksum: checksum,
		Origin:   action.Origin,
	}}, nil
}

// placeDirSymlink creates a single symlink at action.Dest pointing at the
// source directory. The lockfile records one entry; remove uses os.Remove
// which works on symlinks regardless of what they target.
func placeDirSymlink(action LinkAction) ([]LinkResult, error) {
	if err := os.MkdirAll(filepath.Dir(action.Dest), 0o755); err != nil {
		return nil, fmt.Errorf("create parent dir for %s: %w", action.Dest, err)
	}
	if err := os.Symlink(action.Src, action.Dest); err != nil {
		return nil, fmt.Errorf("symlink %s -> %s: %w", action.Dest, action.Src, err)
	}
	return []LinkResult{{
		Src:    action.Src,
		Dest:   action.Dest,
		Method: "symlink-dir",
		Origin: action.Origin,
	}}, nil
}

// placeDirCopy walks action.Src and emits one LinkResult per regular file
// leaf, copying each under action.Dest with sha256 checksum. The walk:
//
//   - Skips symlinks within the source tree (mirrors extractor.go's policy
//     against archive symlinks; see safeJoin in extractor.go).
//   - Honors action.Exclude via matchExclude — directory matches return
//     filepath.SkipDir, file matches are simply skipped.
//   - Sanity-checks the relative path so a symlink-pointed-elsewhere can't
//     escape action.Dest. The archive-side analogue is safeJoin.
func placeDirCopy(action LinkAction) ([]LinkResult, error) {
	var results []LinkResult
	err := filepath.WalkDir(action.Src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(action.Src, path)
		if err != nil {
			return err
		}
		// Mirrors extractor.go:safeJoin — refuse anything that would escape.
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(rel, "..") {
			return fmt.Errorf("copy %s: walked path escapes source: %s", action.Src, rel)
		}

		// Type-route. Symlinks within the source are skipped intentionally;
		// expanding them risks following arbitrary user data into the dest.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		if d.IsDir() {
			if MatchExclude(rel, action.Exclude) {
				return filepath.SkipDir
			}
			return nil
		}

		// Regular file (or device node, etc.). Skip non-regular files to
		// match extractor.go which only writes regular files.
		if !d.Type().IsRegular() {
			return nil
		}
		if MatchExclude(rel, action.Exclude) {
			return nil
		}

		dest := filepath.Join(action.Dest, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("create parent dir for %s: %w", dest, err)
		}
		cs, err := copyFile(path, dest)
		if err != nil {
			return fmt.Errorf("copy %s -> %s: %w", path, dest, err)
		}
		results = append(results, LinkResult{
			Src:      path,
			Dest:     dest,
			Method:   "copy-dir-leaf",
			Checksum: cs,
			Origin:   action.Origin,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// MatchExclude reports whether rel matches any of the supplied glob patterns
// using path/filepath.Match semantics. A pattern matches when it equals the
// full source-relative path (`a/b/c.py`) OR any single path segment along
// the way (`a`, `b`, `c.py`). The segment rule is what makes a pattern like
// `__pycache__` match deeply nested cache dirs without forcing users to
// write `**/__pycache__`.
//
// The glob library is path/filepath.Match for stdlib parity; doublestar is
// not required because the segment-OR-full rule already covers the common
// "exclude this directory anywhere" case.
//
// Exported so dotsctl's dry-run preview can mirror placeDirCopy's filter
// exactly without re-implementing the rule.
func MatchExclude(rel string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	relSlash := filepath.ToSlash(rel)
	segments := strings.Split(relSlash, "/")
	for _, pat := range patterns {
		if ok, _ := filepath.Match(pat, relSlash); ok {
			return true
		}
		for _, seg := range segments {
			if ok, _ := filepath.Match(pat, seg); ok {
				return true
			}
		}
	}
	return false
}

// RemoveLink removes a placed link/copy at the given path.
func RemoveLink(path string) error {
	return os.Remove(path)
}

// PruneEmptyParents removes empty parent directories of leaf upward, stopping
// at (and not removing) stopAt. ENOTEMPTY (and ENOTDIR / ErrNotExist) are
// silently ignored — siblings from another package or the user must keep
// the directory alive. Used by Remove for "copy-dir-leaf" cleanup so that
// uninstalling a directory-copy package doesn't leave a fossil tree.
func PruneEmptyParents(leaf, stopAt string) {
	dir := filepath.Dir(leaf)
	for {
		if dir == stopAt || dir == "." || dir == string(filepath.Separator) {
			return
		}
		if err := os.Remove(dir); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				dir = filepath.Dir(dir)
				continue
			}
			// ENOTEMPTY or anything else — stop quietly. Best-effort cleanup.
			return
		}
		dir = filepath.Dir(dir)
	}
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
