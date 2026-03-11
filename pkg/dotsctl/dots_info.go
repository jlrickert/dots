package dotsctl

import (
	"context"
	"fmt"

	"github.com/jlrickert/dots/pkg/dots"
)

// InfoResult holds detailed information about a package.
type InfoResult struct {
	Package   string
	Tap       string
	Version   string
	Manifest  *dots.ResolvedManifest
	Installed *dots.InstalledPackage
	Platform  dots.Platform
}

// Info returns detailed information about a package.
func (d *Dots) Info(ctx context.Context, pkgRef string) (*InfoResult, error) {
	tap, pkg, err := splitPackageRef(pkgRef)
	if err != nil {
		return nil, err
	}

	result := &InfoResult{
		Package:  pkgRef,
		Tap:      tap,
		Platform: d.PathService.Platform,
	}

	// Read manifest if available
	manifestData, err := d.readManifest(ctx, tap, pkg)
	if err == nil {
		manifest, err := dots.ParseManifest(manifestData)
		if err == nil {
			resolved := dots.ResolveManifest(manifest, d.PathService.Platform)
			result.Manifest = resolved
			result.Version = resolved.Package.Version
		}
	}

	// Check installed state
	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err == nil {
		for i, p := range lockfile.Installed {
			if p.Package == pkgRef {
				result.Installed = &lockfile.Installed[i]
				break
			}
		}
	}

	return result, nil
}

// Diff shows differences between source and installed files for a package.
type DiffEntry struct {
	File   string
	Status string // "changed", "missing", "extra"
}

// Diff compares source files with installed files for a package.
func (d *Dots) Diff(ctx context.Context, pkgRef string) ([]DiffEntry, error) {
	tap, pkg, err := splitPackageRef(pkgRef)
	if err != nil {
		return nil, err
	}

	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err != nil {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}

	var installed *dots.InstalledPackage
	for i, p := range lockfile.Installed {
		if p.Package == pkgRef {
			installed = &lockfile.Installed[i]
			break
		}
	}
	if installed == nil {
		return nil, &dots.PackageNotFoundError{Tap: tap, Package: pkg}
	}

	var diffs []DiffEntry
	for _, f := range installed.Files {
		if f.Method != "copy" {
			continue
		}
		srcChecksum, err := dots.FileChecksum(f.Src)
		if err != nil {
			diffs = append(diffs, DiffEntry{File: f.Dest, Status: "missing"})
			continue
		}
		destChecksum, err := dots.FileChecksum(f.Dest)
		if err != nil {
			diffs = append(diffs, DiffEntry{File: f.Dest, Status: "missing"})
			continue
		}
		if srcChecksum != destChecksum {
			diffs = append(diffs, DiffEntry{File: f.Dest, Status: "changed"})
		}
	}

	return diffs, nil
}

// Which identifies which package placed a given file.
func (d *Dots) Which(ctx context.Context, filePath string) (string, error) {
	lockfile, err := d.Repo.ReadLockfile(ctx)
	if err != nil {
		return "", fmt.Errorf("read lockfile: %w", err)
	}

	for _, pkg := range lockfile.Installed {
		for _, f := range pkg.Files {
			if f.Dest == filePath {
				return pkg.Package, nil
			}
		}
	}

	return "", fmt.Errorf("file %q not managed by dots", filePath)
}
