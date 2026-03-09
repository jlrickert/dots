package dotsctl

import (
	"context"
	"strings"

	"github.com/jlrickert/dots/pkg/dots"
)

// SearchResult holds a matching package and its metadata.
type SearchResult struct {
	Tap         string
	Name        string
	Description string
	Version     string
	Tags        []string
	Installed   bool
}

// Search finds packages whose name, description, or tags match the query.
func (d *Dots) Search(ctx context.Context, query string) ([]SearchResult, error) {
	taps, err := d.Repo.ListTaps(ctx)
	if err != nil {
		return nil, err
	}

	// Build installed set for marking results.
	installed := make(map[string]bool)
	if lf, err := d.Repo.ReadLockfile(ctx); err == nil {
		for _, pkg := range lf.Installed {
			installed[pkg.Package] = true
		}
	}

	q := strings.ToLower(query)
	var results []SearchResult

	for _, tap := range taps {
		pkgs, err := d.Repo.ListPackages(ctx, tap.Name)
		if err != nil {
			continue
		}
		for _, pkg := range pkgs {
			manifest, err := d.readAndParseManifest(ctx, tap.Name, pkg.Name)
			if err != nil {
				// If manifest can't be parsed, match on name only.
				if matchesQuery(pkg.Name, "", nil, q) {
					ref := tap.Name + "/" + pkg.Name
					results = append(results, SearchResult{
						Tap:       tap.Name,
						Name:      pkg.Name,
						Installed: installed[ref],
					})
				}
				continue
			}

			if matchesQuery(manifest.Package.Name, manifest.Package.Description, manifest.Package.Tags, q) {
				ref := tap.Name + "/" + pkg.Name
				results = append(results, SearchResult{
					Tap:         tap.Name,
					Name:        pkg.Name,
					Description: manifest.Package.Description,
					Version:     manifest.Package.Version,
					Tags:        manifest.Package.Tags,
					Installed:   installed[ref],
				})
			}
		}
	}

	return results, nil
}

func (d *Dots) readAndParseManifest(ctx context.Context, tap, pkg string) (*dots.Manifest, error) {
	data, err := d.Repo.ReadManifest(ctx, tap, pkg)
	if err != nil {
		return nil, err
	}
	return dots.ParseManifest(data)
}

func matchesQuery(name, description string, tags []string, query string) bool {
	if strings.Contains(strings.ToLower(name), query) {
		return true
	}
	if strings.Contains(strings.ToLower(description), query) {
		return true
	}
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}
