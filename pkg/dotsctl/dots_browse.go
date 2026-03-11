package dotsctl

import (
	"context"
)

// BrowseResult holds package listing for a tap or package detail.
type BrowseResult struct {
	Tap      string
	Packages []BrowsePackage
}

// BrowsePackage holds display info for a single package.
type BrowsePackage struct {
	Name        string
	Description string
	Version     string
	Tags        []string
	Installed   bool
	LinkCount   int
}

// Browse lists packages in a tap with their metadata.
func (d *Dots) Browse(ctx context.Context, tap string) (*BrowseResult, error) {
	pkgs, err := d.listPackages(ctx, tap)
	if err != nil {
		return nil, err
	}

	// Build installed set.
	installed := make(map[string]bool)
	if lf, err := d.Repo.ReadLockfile(ctx); err == nil {
		for _, pkg := range lf.Installed {
			installed[pkg.Package] = true
		}
	}

	result := &BrowseResult{Tap: tap}

	for _, pkg := range pkgs {
		bp := BrowsePackage{
			Name:      pkg.Name,
			Installed: installed[tap+"/"+pkg.Name],
		}

		manifest, err := d.readAndParseManifest(ctx, tap, pkg.Name)
		if err == nil {
			bp.Description = manifest.Package.Description
			bp.Version = manifest.Package.Version
			bp.Tags = manifest.Package.Tags
			bp.LinkCount = len(manifest.Links)
		}

		result.Packages = append(result.Packages, bp)
	}

	return result, nil
}
