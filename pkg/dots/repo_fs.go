package dots

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

// FsRepo is a filesystem-backed Repository implementation.
// Taps are stored as cloned repos under StateDir/taps/<name>/,
// the lockfile lives at StateDir/dots.lock.yaml, and backups
// are stored under StateDir/backups/.
type FsRepo struct {
	ConfigDir string // e.g. ~/.config/dots
	StateDir  string // e.g. ~/.local/state/dots
	Git       GitClient

	mu sync.RWMutex
}

// NewFsRepo creates a filesystem-backed repository.
// If git is nil, ExecGitClient is used.
func NewFsRepo(configDir, stateDir string, git GitClient) *FsRepo {
	if git == nil {
		git = NewExecGitClient()
	}
	return &FsRepo{
		ConfigDir: configDir,
		StateDir:  stateDir,
		Git:       git,
	}
}

func (r *FsRepo) Name() string { return "fs" }

// tapsDir returns the directory where tap clones are stored.
func (r *FsRepo) tapsDir() string {
	return filepath.Join(r.StateDir, "taps")
}

// tapDir returns the directory for a specific tap.
func (r *FsRepo) tapDir(name string) string {
	return filepath.Join(r.tapsDir(), name)
}

// lockfilePath returns the path to dots.lock.yaml.
func (r *FsRepo) lockfilePath() string {
	return filepath.Join(r.StateDir, "dots.lock.yaml")
}

// backupsDir returns the directory for file backups.
func (r *FsRepo) backupsDir() string {
	return filepath.Join(r.StateDir, "backups")
}

// tapInfoPath returns the path to tap metadata file within a tap directory.
func (r *FsRepo) tapInfoPath(name string) string {
	return filepath.Join(r.tapDir(name), ".dots-tap.yaml")
}

// --- Tap management ---

func (r *FsRepo) ListTaps(ctx context.Context) ([]TapInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, err := os.ReadDir(r.tapsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list taps: %w", err)
	}

	var taps []TapInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := r.readTapInfo(entry.Name())
		if err != nil {
			continue // skip taps with invalid metadata
		}
		taps = append(taps, *info)
	}

	sort.Slice(taps, func(i, j int) bool {
		return taps[i].Name < taps[j].Name
	})
	return taps, nil
}

func (r *FsRepo) GetTap(ctx context.Context, name string) (*TapInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.readTapInfo(name)
}

func (r *FsRepo) AddTap(ctx context.Context, tap TapInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dir := r.tapDir(tap.Name)
	if _, err := os.Stat(dir); err == nil {
		return ErrExist
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(r.tapsDir(), 0o755); err != nil {
		return fmt.Errorf("create taps dir: %w", err)
	}

	// Clone the repository into the tap directory.
	err := r.Git.Clone(ctx, tap.URL, dir, GitCloneOpts{
		Branch: tap.Branch,
	})
	if err != nil {
		// Clean up partial clone on failure.
		os.RemoveAll(dir)
		return fmt.Errorf("clone tap %s: %w", tap.Name, err)
	}

	return r.writeTapInfo(&tap)
}

func (r *FsRepo) RemoveTap(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dir := r.tapDir(name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return &TapNotFoundError{Name: name}
	}

	return os.RemoveAll(dir)
}

func (r *FsRepo) UpdateTap(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dir := r.tapDir(name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return &TapNotFoundError{Name: name}
	}

	if err := r.Git.Pull(ctx, dir); err != nil {
		return fmt.Errorf("update tap %s: %w", name, err)
	}
	return nil
}

// --- Package discovery ---

func (r *FsRepo) ListPackages(ctx context.Context, tap string) ([]PackageInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dir := r.tapDir(tap)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, &TapNotFoundError{Name: tap}
	}

	return ScanPackages(tap, dir)
}

// ScanPackages walks a directory for Dotfile.yaml files and returns discovered packages.
func ScanPackages(tap, dir string) ([]PackageInfo, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, &TapNotFoundError{Name: tap}
	}

	var packages []PackageInfo
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "Dotfile.yaml" {
			return nil
		}

		pkgDir := filepath.Dir(path)
		relDir, err := filepath.Rel(dir, pkgDir)
		if err != nil {
			return err
		}

		// Skip the tap metadata directory
		if relDir == "." {
			return nil
		}

		// The package name is the directory name containing Dotfile.yaml
		name := filepath.Base(pkgDir)

		packages = append(packages, PackageInfo{
			Tap:  tap,
			Name: name,
			Dir:  relDir,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk tap %s: %w", tap, err)
	}

	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Name < packages[j].Name
	})
	return packages, nil
}

func (r *FsRepo) ReadManifest(ctx context.Context, tap, pkg string) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dir := r.tapDir(tap)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, &TapNotFoundError{Name: tap}
	}

	// Try direct path first: taps/<tap>/<pkg>/Dotfile.yaml
	manifestPath := filepath.Join(dir, pkg, "Dotfile.yaml")
	data, err := os.ReadFile(manifestPath)
	if err == nil {
		return data, nil
	}

	// Walk to find the package if not at the direct path
	var found []byte
	walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() != "Dotfile.yaml" {
			return nil
		}
		if filepath.Base(filepath.Dir(path)) == pkg {
			found, err = os.ReadFile(path)
			return err
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("search for package %s/%s: %w", tap, pkg, walkErr)
	}
	if found == nil {
		return nil, &PackageNotFoundError{Tap: tap, Package: pkg}
	}
	return found, nil
}

// --- Lockfile ---

func (r *FsRepo) ReadLockfile(ctx context.Context) (*Lockfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := os.ReadFile(r.lockfilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotExist
		}
		return nil, fmt.Errorf("read lockfile: %w", err)
	}

	var lf Lockfile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("%w: lockfile: %v", ErrParse, err)
	}
	return &lf, nil
}

func (r *FsRepo) WriteLockfile(ctx context.Context, lockfile *Lockfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := os.MkdirAll(r.StateDir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	data, err := yaml.Marshal(lockfile)
	if err != nil {
		return fmt.Errorf("marshal lockfile: %w", err)
	}

	return os.WriteFile(r.lockfilePath(), data, 0o644)
}

// --- Backups ---

func (r *FsRepo) BackupFile(ctx context.Context, path string, data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	backupPath := filepath.Join(r.backupsDir(), path)
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}

	return os.WriteFile(backupPath, data, 0o644)
}

func (r *FsRepo) RestoreFile(ctx context.Context, path string) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	backupPath := filepath.Join(r.backupsDir(), path)
	data, err := os.ReadFile(backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotExist
		}
		return nil, fmt.Errorf("restore file: %w", err)
	}
	return data, nil
}

func (r *FsRepo) ListBackups(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	backupsDir := r.backupsDir()
	if _, err := os.Stat(backupsDir); os.IsNotExist(err) {
		return nil, nil
	}

	var paths []string
	err := filepath.WalkDir(backupsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(backupsDir, path)
		if err != nil {
			return err
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}

	sort.Strings(paths)
	return paths, nil
}

// --- Internal helpers ---

func (r *FsRepo) readTapInfo(name string) (*TapInfo, error) {
	data, err := os.ReadFile(r.tapInfoPath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &TapNotFoundError{Name: name}
		}
		return nil, fmt.Errorf("read tap info: %w", err)
	}

	var info TapInfo
	if err := yaml.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("%w: tap info: %v", ErrParse, err)
	}
	info.Name = name
	return &info, nil
}

func (r *FsRepo) writeTapInfo(info *TapInfo) error {
	data, err := yaml.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal tap info: %w", err)
	}
	return os.WriteFile(r.tapInfoPath(info.Name), data, 0o644)
}

// Compile-time interface check.
var _ Repository = (*FsRepo)(nil)
