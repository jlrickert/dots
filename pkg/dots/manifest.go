package dots

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

const (
	// DotfileSchemaURL is the URL for the Dotfile.yaml JSON Schema.
	DotfileSchemaURL = "https://raw.githubusercontent.com/jlrickert/dots/main/schemas/dotfile.json"
	// DotfileSchemaModeline is the yaml-language-server modeline for Dotfile.yaml.
	DotfileSchemaModeline = "# yaml-language-server: $schema=" + DotfileSchemaURL + "\n"
)

// Manifest represents a parsed Dotfile.yaml package manifest.
type Manifest struct {
	Package  ManifestPackage           `yaml:"package"`
	Links    map[string]string         `yaml:"links,omitempty"`
	Hooks    ManifestHooks             `yaml:"hooks,omitempty"`
	Overlay  *ManifestOverlay          `yaml:"overlay,omitempty"`
	Merge    map[string]string         `yaml:"merge,omitempty"`
	Platform map[string]PlatformBlock  `yaml:"platform,omitempty"`
}

// ManifestPackage is the `package:` section of a manifest.
type ManifestPackage struct {
	Name         string       `yaml:"name"`
	Description  string       `yaml:"description,omitempty"`
	Version      string       `yaml:"version,omitempty"`
	Requires     []string     `yaml:"requires,omitempty"`
	Tags         []string     `yaml:"tags,omitempty"`
	Platforms    []string     `yaml:"platforms,omitempty"`
	LinkStrategy LinkStrategy `yaml:"link_strategy,omitempty"`
}

// ManifestHooks holds lifecycle hook script paths.
type ManifestHooks struct {
	PreInstall  string `yaml:"pre_install,omitempty"`
	PostInstall string `yaml:"post_install,omitempty"`
	PreRemove   string `yaml:"pre_remove,omitempty"`
	PostRemove  string `yaml:"post_remove,omitempty"`
	PreUpgrade  string `yaml:"pre_upgrade,omitempty"`
	PostUpgrade string `yaml:"post_upgrade,omitempty"`
}

// ManifestOverlay declares this package as a layer on top of another.
type ManifestOverlay struct {
	Base     string `yaml:"base"`
	Strategy string `yaml:"strategy,omitempty"` // append, prepend, replace, merge
	Priority int    `yaml:"priority,omitempty"` // 0-99
}

// PlatformBlock holds platform-specific overrides within a manifest.
type PlatformBlock struct {
	Links        map[string]string `yaml:"links,omitempty"`
	Hooks        ManifestHooks     `yaml:"hooks,omitempty"`
	Requires     []string          `yaml:"requires,omitempty"`
	Tags         []string          `yaml:"tags,omitempty"`
	Overlay      *ManifestOverlay  `yaml:"overlay,omitempty"`
	Merge        map[string]string `yaml:"merge,omitempty"`
	LinkStrategy LinkStrategy      `yaml:"link_strategy,omitempty"`
}

// ParseManifest parses a Manifest from YAML bytes.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParse, err)
	}
	if m.Package.Name == "" {
		return nil, fmt.Errorf("%w: package.name is required", ErrParse)
	}
	return &m, nil
}

// ResolvedManifest is the effective manifest after platform cascade resolution.
type ResolvedManifest struct {
	Package      ManifestPackage
	Links        map[string]string
	Hooks        ManifestHooks
	Overlay      *ManifestOverlay
	Merge        map[string]string
	LinkStrategy LinkStrategy
}

// ResolveManifest applies platform cascade resolution (base → OS → OS-arch)
// to a manifest for the given platform.
func ResolveManifest(m *Manifest, p Platform) *ResolvedManifest {
	r := &ResolvedManifest{
		Package:      m.Package,
		Links:        copyStringMap(m.Links),
		Hooks:        m.Hooks,
		Overlay:      m.Overlay,
		Merge:        copyStringMap(m.Merge),
		LinkStrategy: m.Package.LinkStrategy,
	}

	// Collect tags and requires from base.
	tags := append([]string{}, m.Package.Tags...)
	requires := append([]string{}, m.Package.Requires...)

	// Apply OS-only section.
	if osBlock, ok := m.Platform[p.OS]; ok {
		applyPlatformBlock(r, &osBlock)
		tags = appendDedup(tags, osBlock.Tags)
		requires = appendDedup(requires, osBlock.Requires)
	}

	// Apply OS-arch section.
	if archBlock, ok := m.Platform[p.String()]; ok {
		applyPlatformBlock(r, &archBlock)
		tags = appendDedup(tags, archBlock.Tags)
		requires = appendDedup(requires, archBlock.Requires)
	}

	r.Package.Tags = tags
	r.Package.Requires = requires
	return r
}

// SupportsCurrentPlatform checks if the manifest's platforms list (if set)
// includes the given platform.
func (m *Manifest) SupportsCurrentPlatform(p Platform) bool {
	if len(m.Package.Platforms) == 0 {
		return true
	}
	for _, supported := range m.Package.Platforms {
		if supported == p.String() || supported == p.OS {
			return true
		}
	}
	return false
}

func applyPlatformBlock(r *ResolvedManifest, block *PlatformBlock) {
	for k, v := range block.Links {
		r.Links[k] = v
	}
	if block.Hooks.PreInstall != "" {
		r.Hooks.PreInstall = block.Hooks.PreInstall
	}
	if block.Hooks.PostInstall != "" {
		r.Hooks.PostInstall = block.Hooks.PostInstall
	}
	if block.Hooks.PreRemove != "" {
		r.Hooks.PreRemove = block.Hooks.PreRemove
	}
	if block.Hooks.PostRemove != "" {
		r.Hooks.PostRemove = block.Hooks.PostRemove
	}
	if block.Hooks.PreUpgrade != "" {
		r.Hooks.PreUpgrade = block.Hooks.PreUpgrade
	}
	if block.Hooks.PostUpgrade != "" {
		r.Hooks.PostUpgrade = block.Hooks.PostUpgrade
	}
	if block.Overlay != nil {
		r.Overlay = block.Overlay
	}
	for k, v := range block.Merge {
		if r.Merge == nil {
			r.Merge = make(map[string]string)
		}
		r.Merge[k] = v
	}
	if block.LinkStrategy != "" {
		r.LinkStrategy = block.LinkStrategy
	}
}

func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return make(map[string]string)
	}
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func appendDedup(base, extra []string) []string {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base))
	for _, s := range base {
		seen[s] = struct{}{}
	}
	for _, s := range extra {
		if _, ok := seen[s]; !ok {
			base = append(base, s)
			seen[s] = struct{}{}
		}
	}
	return base
}
