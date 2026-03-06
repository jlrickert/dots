package dots

// FsRepo is a filesystem-backed Repository implementation.
// It will store taps as git clones, lockfile as dots.lock.yaml,
// and backups in the state directory.
type FsRepo struct {
	ConfigDir string // e.g. ~/.config/dots
	StateDir  string // e.g. ~/.local/state/dots
}

// Placeholder — FsRepo methods will be implemented in Phase 5
// when the service layer wires everything together.
