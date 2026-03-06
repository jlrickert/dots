package dots

import (
	"context"
	"time"
)

// LinkStrategy defines how dots places files at their target location.
type LinkStrategy string

const (
	LinkSymlink  LinkStrategy = "symlink"
	LinkCopy     LinkStrategy = "copy"
	LinkHardlink LinkStrategy = "hardlink"
)

// TapInfo describes a registered tap (pointer to a Git repo).
type TapInfo struct {
	Name       string
	URL        string
	Branch     string
	Provider   string // github, bitbucket, generic
	Visibility string // public, private
}

// PackageInfo describes a discovered package within a tap.
type PackageInfo struct {
	Tap  string
	Name string
	Dir  string // relative path within the tap repo
}

// InstalledFile records a single linked file in the lockfile.
type InstalledFile struct {
	Src      string `yaml:"src"`
	Dest     string `yaml:"dest"`
	Origin   string `yaml:"origin"`   // base, darwin, darwin-arm64, etc.
	Method   string `yaml:"method"`   // symlink, copy, hardlink
	Checksum string `yaml:"checksum"` // sha256:...
}

// InstalledPackage records an installed package in the lockfile.
type InstalledPackage struct {
	Package          string          `yaml:"package"` // tap/name
	Tap              string          `yaml:"tap"`
	Commit           string          `yaml:"commit"`
	Version          string          `yaml:"version"`
	Type             string          `yaml:"type"` // base, overlay
	LinkStrategy     LinkStrategy    `yaml:"link_strategy"`
	PlatformResolved []string        `yaml:"platform_resolved"`
	Files            []InstalledFile `yaml:"files"`
}

// LockfileState is the top-level state section of the lockfile.
type LockfileState struct {
	ActiveProfile string       `yaml:"active_profile"`
	LastApplied   time.Time    `yaml:"last_applied"`
	Platform      string       `yaml:"platform"`
	LinkStrategy  LinkStrategy `yaml:"link_strategy"`
}

// Lockfile represents the full dots.lock.yaml.
type Lockfile struct {
	State     LockfileState      `yaml:"state"`
	Installed []InstalledPackage `yaml:"installed"`
}

// Repository is the storage backend contract for dots. Implementations handle
// tap management, package discovery, lockfile persistence, and backups.
type Repository interface {
	// Name returns a short identifier for the backend (e.g. "memory", "fs").
	Name() string

	// --- Tap management ---

	// ListTaps returns all registered taps.
	ListTaps(ctx context.Context) ([]TapInfo, error)
	// GetTap returns info for a single tap by name.
	GetTap(ctx context.Context, name string) (*TapInfo, error)
	// AddTap registers a new tap.
	AddTap(ctx context.Context, tap TapInfo) error
	// RemoveTap removes a registered tap and its cloned data.
	RemoveTap(ctx context.Context, name string) error
	// UpdateTap fetches the latest state for a tap (e.g. git pull).
	UpdateTap(ctx context.Context, name string) error

	// --- Package discovery ---

	// ListPackages returns all packages available in a tap.
	ListPackages(ctx context.Context, tap string) ([]PackageInfo, error)
	// ReadManifest returns the raw Dotfile.yaml bytes for a package.
	ReadManifest(ctx context.Context, tap, pkg string) ([]byte, error)

	// --- Lockfile ---

	// ReadLockfile returns the current lockfile state.
	ReadLockfile(ctx context.Context) (*Lockfile, error)
	// WriteLockfile persists the lockfile state.
	WriteLockfile(ctx context.Context, lockfile *Lockfile) error

	// --- Backups ---

	// BackupFile stores a backup of a file before it is replaced.
	BackupFile(ctx context.Context, path string, data []byte) error
	// RestoreFile retrieves a previously backed-up file.
	RestoreFile(ctx context.Context, path string) ([]byte, error)
	// ListBackups returns all backed-up file paths.
	ListBackups(ctx context.Context) ([]string, error)
}
