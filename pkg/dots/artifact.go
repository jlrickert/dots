package dots

// Artifact describes a single binary or tarball to fetch and (optionally)
// extract before linking. Manifests embed a slice of Artifact under the
// `artifacts:` block. Each entry is content-addressed by Sha256: the fetcher
// refuses to surface bytes that do not match, so corruption and source drift
// surface as errors rather than silent installs.
//
// Stage names the on-disk identity: extracted bytes land under
// @artifacts/<tap>/<pkg>/<Stage>/, which is also the path prefix manifest
// link entries use to point into the staged tree. Stage is required when
// Extract is anything other than ExtractNone; for raw downloads it defaults
// to the basename of URL.
type Artifact struct {
	URL             string        `yaml:"url"`
	Sha256          string        `yaml:"sha256"`
	Extract         ExtractFormat `yaml:"extract,omitempty"`
	StripComponents int           `yaml:"strip_components,omitempty"`
	Stage           string        `yaml:"stage,omitempty"`
	// Executables lists POSIX-executable paths within the staged tree. The
	// manifest loader (Phase 2) converts this slice to a set at the
	// ExtractRequest boundary; the on-disk shape is a list because YAML
	// round-trips ordered sequences cleanly.
	Executables []string `yaml:"executables,omitempty"`
}

// ExtractFormat enumerates supported archive shapes. Unknown values surface
// as ErrUnsupportedExtract from the extractor; the parser is permissive so
// that future formats can be added without a breaking schema change.
type ExtractFormat string

const (
	// ExtractNone keeps the downloaded bytes as a single staged file. This is
	// the right choice for prebuilt single-binary releases (e.g. a `dots`
	// binary published directly on a GitHub release).
	ExtractNone ExtractFormat = ""
	// ExtractTarGz unpacks a gzip-compressed tarball.
	ExtractTarGz ExtractFormat = "tar.gz"
	// ExtractTarXz unpacks an xz-compressed tarball.
	ExtractTarXz ExtractFormat = "tar.xz"
	// ExtractZip unpacks a zip archive.
	ExtractZip ExtractFormat = "zip"
)
