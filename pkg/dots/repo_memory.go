package dots

import (
	"context"
	"sort"
	"sync"
)

// MemoryRepo is an in-memory Repository implementation for unit tests.
type MemoryRepo struct {
	mu       sync.RWMutex
	taps     map[string]*TapInfo
	packages map[string][]memoryPackage // keyed by tap name
	lockfile *Lockfile
	backups  map[string][]byte
}

type memoryPackage struct {
	info     PackageInfo
	manifest []byte
}

// NewMemoryRepo creates a ready-to-use in-memory repository.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		taps:     make(map[string]*TapInfo),
		packages: make(map[string][]memoryPackage),
		backups:  make(map[string][]byte),
	}
}

func (r *MemoryRepo) Name() string { return "memory" }

// --- Tap management ---

func (r *MemoryRepo) ListTaps(ctx context.Context) ([]TapInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]TapInfo, 0, len(r.taps))
	for _, t := range r.taps {
		result = append(result, *t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (r *MemoryRepo) GetTap(ctx context.Context, name string) (*TapInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.taps[name]
	if !ok {
		return nil, &TapNotFoundError{Name: name}
	}
	cp := *t
	return &cp, nil
}

func (r *MemoryRepo) AddTap(ctx context.Context, tap TapInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.taps[tap.Name]; ok {
		return ErrExist
	}
	cp := tap
	r.taps[tap.Name] = &cp
	return nil
}

func (r *MemoryRepo) RemoveTap(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.taps[name]; !ok {
		return &TapNotFoundError{Name: name}
	}
	delete(r.taps, name)
	delete(r.packages, name)
	return nil
}

func (r *MemoryRepo) UpdateTap(ctx context.Context, name string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.taps[name]; !ok {
		return &TapNotFoundError{Name: name}
	}
	// No-op for memory backend — nothing to fetch.
	return nil
}

// --- Package discovery ---

func (r *MemoryRepo) ListPackages(ctx context.Context, tap string) ([]PackageInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.taps[tap]; !ok {
		return nil, &TapNotFoundError{Name: tap}
	}
	pkgs := r.packages[tap]
	result := make([]PackageInfo, len(pkgs))
	for i, p := range pkgs {
		result[i] = p.info
	}
	return result, nil
}

func (r *MemoryRepo) ReadManifest(ctx context.Context, tap, pkg string) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.taps[tap]; !ok {
		return nil, &TapNotFoundError{Name: tap}
	}
	for _, p := range r.packages[tap] {
		if p.info.Name == pkg {
			cp := make([]byte, len(p.manifest))
			copy(cp, p.manifest)
			return cp, nil
		}
	}
	return nil, &PackageNotFoundError{Tap: tap, Package: pkg}
}

// AddPackage is a test helper to seed a package into a tap.
func (r *MemoryRepo) AddPackage(tap string, info PackageInfo, manifest []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.taps[tap]; !ok {
		return &TapNotFoundError{Name: tap}
	}
	r.packages[tap] = append(r.packages[tap], memoryPackage{
		info:     info,
		manifest: manifest,
	})
	return nil
}

// --- Lockfile ---

func (r *MemoryRepo) ReadLockfile(ctx context.Context) (*Lockfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.lockfile == nil {
		return nil, ErrNotExist
	}
	cp := *r.lockfile
	cp.Installed = make([]InstalledPackage, len(r.lockfile.Installed))
	copy(cp.Installed, r.lockfile.Installed)
	return &cp, nil
}

func (r *MemoryRepo) WriteLockfile(ctx context.Context, lockfile *Lockfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *lockfile
	cp.Installed = make([]InstalledPackage, len(lockfile.Installed))
	copy(cp.Installed, lockfile.Installed)
	r.lockfile = &cp
	return nil
}

// --- Backups ---

func (r *MemoryRepo) BackupFile(ctx context.Context, path string, data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	r.backups[path] = cp
	return nil
}

func (r *MemoryRepo) RestoreFile(ctx context.Context, path string) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, ok := r.backups[path]
	if !ok {
		return nil, ErrNotExist
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, nil
}

func (r *MemoryRepo) ListBackups(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	paths := make([]string, 0, len(r.backups))
	for p := range r.backups {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}

// Compile-time interface check.
var _ Repository = (*MemoryRepo)(nil)
