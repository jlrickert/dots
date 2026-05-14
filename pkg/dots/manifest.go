package dots

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DotfileSchemaURL is the URL for the Dotfile.yaml JSON Schema.
	DotfileSchemaURL = "https://raw.githubusercontent.com/jlrickert/dots/main/schemas/dotfile.json"
	// DotfileSchemaModeline is the yaml-language-server modeline for Dotfile.yaml.
	// Dots does not currently scaffold Dotfile.yaml files; this constant is the
	// canonical string for any future package-scaffolding command and for
	// authors hand-writing manifests.
	DotfileSchemaModeline = "# yaml-language-server: $schema=" + DotfileSchemaURL + "\n"
	// SelfTapPrefix is the pseudo-prefix that, in package refs inside a
	// manifest (requires, overlay.base), resolves to the tap the manifest
	// was loaded from. It lets authors write portable intra-tap references
	// that don't depend on the consumer's tap alias.
	SelfTapPrefix = "@self/"
)

// LinkMode discriminates how a directory link entry is materialized at the
// destination. The empty value (LinkModeAuto) defers to the resolved
// LinkStrategy at link time. For file sources the mode is ignored — files
// follow the package's LinkStrategy as before.
type LinkMode string

const (
	// LinkModeAuto is the default mode. For directory sources it resolves to
	// symlink or copy based on the resolved LinkStrategy at link time.
	LinkModeAuto LinkMode = ""
	// LinkModeSymlink forces a single symlink at the directory root.
	LinkModeSymlink LinkMode = "symlink"
	// LinkModeCopy forces a recursive per-leaf copy.
	LinkModeCopy LinkMode = "copy"
)

// LinkSpec is a single entry in a manifest's links: map. It accepts two YAML
// shapes:
//
//   - String shorthand: `src: target` parses as LinkSpec{Target: target,
//     Mode: LinkModeAuto}. This is the historical shape and remains the
//     dominant form. Bare-string entries that resolve to a directory at link
//     time auto-symlink (or auto-copy, depending on LinkStrategy).
//   - Object form: `src: {target: ..., mode: symlink|copy|auto, exclude: [glob, ...]}`.
//     Use this when a directory needs explicit copy semantics with excludes,
//     or when documenting the intent of an auto-symlink-on-directory entry.
//
// Unknown modes are rejected at parse time as ErrParse.
type LinkSpec struct {
	// Target is the destination path. Supports @alias prefixes and raw
	// home-relative paths. Required for both shapes.
	Target string `yaml:"target"`
	// Mode is the link discriminator; empty means LinkModeAuto.
	Mode LinkMode `yaml:"mode,omitempty"`
	// Exclude is a list of filepath.Match globs applied during recursive
	// directory copies (LinkModeCopy on directory sources). Patterns match
	// against any path segment or the full source-relative path; document the
	// rule in matchExclude in linker.go.
	Exclude []string `yaml:"exclude,omitempty"`
}

// UnmarshalYAML implements custom decoding for LinkSpec so the links: map
// supports both the string shorthand and the object form. Anything else is
// rejected as ErrParse with the source location attached.
func (s *LinkSpec) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// String shorthand. Mode is left as LinkModeAuto so the linker can
		// resolve it once the source's directory-ness is known.
		s.Target = node.Value
		s.Mode = LinkModeAuto
		return nil
	case yaml.MappingNode:
		// Use a private alias to avoid recursion through this UnmarshalYAML.
		type rawLinkSpec LinkSpec
		var raw rawLinkSpec
		if err := node.Decode(&raw); err != nil {
			return fmt.Errorf("%w: link entry at line %d: %v", ErrParse, node.Line, err)
		}
		if raw.Target == "" {
			return fmt.Errorf("%w: link entry at line %d: target is required", ErrParse, node.Line)
		}
		switch raw.Mode {
		case LinkModeAuto, LinkModeSymlink, LinkModeCopy:
			// ok
		default:
			return fmt.Errorf("%w: link entry at line %d: invalid mode %q (want auto, symlink, or copy)", ErrParse, node.Line, raw.Mode)
		}
		*s = LinkSpec(raw)
		return nil
	default:
		return fmt.Errorf("%w: link entry at line %d: expected string or mapping", ErrParse, node.Line)
	}
}

// Manifest represents a parsed Dotfile.yaml package manifest.
type Manifest struct {
	Package  ManifestPackage          `yaml:"package"`
	Links    map[string]LinkSpec      `yaml:"links,omitempty"`
	Hooks    ManifestHooks            `yaml:"hooks,omitempty"`
	Overlay  *ManifestOverlay         `yaml:"overlay,omitempty"`
	Merge    map[string]string        `yaml:"merge,omitempty"`
	Platform map[string]PlatformBlock `yaml:"platform,omitempty"`
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
	Links        map[string]LinkSpec `yaml:"links,omitempty"`
	Hooks        ManifestHooks       `yaml:"hooks,omitempty"`
	Requires     []string            `yaml:"requires,omitempty"`
	Tags         []string            `yaml:"tags,omitempty"`
	Overlay      *ManifestOverlay    `yaml:"overlay,omitempty"`
	Merge        map[string]string   `yaml:"merge,omitempty"`
	LinkStrategy LinkStrategy        `yaml:"link_strategy,omitempty"`
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
	Links        map[string]LinkSpec
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
		Links:        copyLinkMap(m.Links),
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

// copyLinkMap returns a deep copy of a links map. LinkSpec.Exclude is
// duplicated so callers can safely mutate the result without aliasing the
// source manifest's slices.
func copyLinkMap(m map[string]LinkSpec) map[string]LinkSpec {
	if m == nil {
		return make(map[string]LinkSpec)
	}
	cp := make(map[string]LinkSpec, len(m))
	for k, v := range m {
		spec := LinkSpec{Target: v.Target, Mode: v.Mode}
		if len(v.Exclude) > 0 {
			spec.Exclude = append([]string(nil), v.Exclude...)
		}
		cp[k] = spec
	}
	return cp
}

// ResolveSelfRef rewrites a single package reference, expanding the
// "@self/" pseudo-prefix to currentTap. Refs without the prefix pass through
// unchanged. Using "@self/" with an empty currentTap is an error.
func ResolveSelfRef(ref, currentTap string) (string, error) {
	if !strings.HasPrefix(ref, SelfTapPrefix) {
		return ref, nil
	}
	if currentTap == "" {
		return "", fmt.Errorf("%w: cannot resolve %q without a current tap", ErrParse, ref)
	}
	return currentTap + "/" + strings.TrimPrefix(ref, SelfTapPrefix), nil
}

// ResolveSelfRefs rewrites "@self/" pseudo-prefixes in Requires and
// Overlay.Base using currentTap. Safe to call more than once; non-self refs
// pass through unchanged.
func (r *ResolvedManifest) ResolveSelfRefs(currentTap string) error {
	for i, ref := range r.Package.Requires {
		resolved, err := ResolveSelfRef(ref, currentTap)
		if err != nil {
			return err
		}
		r.Package.Requires[i] = resolved
	}
	if r.Overlay != nil {
		resolved, err := ResolveSelfRef(r.Overlay.Base, currentTap)
		if err != nil {
			return err
		}
		if resolved != r.Overlay.Base {
			// Detach from the source manifest before mutating: ResolveManifest
			// stores the *ManifestOverlay by pointer from the source Manifest.
			clone := *r.Overlay
			clone.Base = resolved
			r.Overlay = &clone
		}
	}
	return nil
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
