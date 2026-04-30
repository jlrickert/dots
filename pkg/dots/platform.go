package dots

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jlrickert/cli-toolkit/toolkit"
)

// Platform represents an OS-architecture pair (e.g. "darwin-arm64").
type Platform struct {
	OS   string // darwin, linux, windows, freebsd
	Arch string // amd64, arm64
}

// String returns the platform identifier in "os-arch" format.
func (p Platform) String() string {
	return p.OS + "-" + p.Arch
}

// DetectPlatform returns the current platform from Go runtime.
func DetectPlatform() Platform {
	return Platform{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

// Alias is a named path shorthand (e.g. "@config") that resolves to a
// platform-native directory.
type Alias = string

const (
	AliasHome   Alias = "@home"
	AliasConfig Alias = "@config"
	AliasData   Alias = "@data"
	AliasCache  Alias = "@cache"
	AliasState  Alias = "@state"
	AliasBin    Alias = "@bin"

	// XDG family — always resolves to XDG paths regardless of OS, including
	// Windows. Honors XDG_*_HOME env vars, otherwise falls back to the
	// standard XDG default directories (~/.config, ~/.local/share, ~/.cache,
	// ~/.local/state). XDG does not specify a binary directory, so there is
	// no @xdg-bin.
	AliasXDGConfig Alias = "@xdg-config"
	AliasXDGData   Alias = "@xdg-data"
	AliasXDGCache  Alias = "@xdg-cache"
	AliasXDGState  Alias = "@xdg-state"

	// Apple family — Apple HIG-style locations under ~/Library on darwin
	// only. On non-darwin platforms these aliases return an
	// AliasUnavailableError wrapping ErrAliasUnavailable. Note that
	// @apple-config and @apple-data both resolve to
	// ~/Library/Application Support; Apple HIG conflates these.
	AliasAppleConfig       Alias = "@apple-config"
	AliasAppleData         Alias = "@apple-data"
	AliasAppleCache        Alias = "@apple-cache"
	AliasAppleLogs         Alias = "@apple-logs"
	AliasAppleLaunchAgents Alias = "@apple-launchagents"
)

// BuiltinAliases is the ordered list of built-in alias names.
var BuiltinAliases = []Alias{
	AliasHome, AliasConfig, AliasData, AliasCache, AliasState, AliasBin,
	AliasXDGConfig, AliasXDGData, AliasXDGCache, AliasXDGState,
	AliasAppleConfig, AliasAppleData, AliasAppleCache, AliasAppleLogs, AliasAppleLaunchAgents,
}

// AliasResolver resolves path aliases to absolute paths for a given platform.
type AliasResolver struct {
	platform Platform
	env      toolkit.Env
	home     string
	custom   map[string]string
}

// NewAliasResolver creates a resolver for the given platform. The env is used
// to look up environment variables and the user's home directory.
func NewAliasResolver(p Platform, env toolkit.Env) *AliasResolver {
	home, _ := env.GetHome()
	return &AliasResolver{
		platform: p,
		env:      env,
		home:     home,
	}
}

// SetCustomAliases registers user-defined aliases from config. Custom aliases
// may reference built-in aliases (e.g. "@dots": "@config/dots").
func (r *AliasResolver) SetCustomAliases(aliases map[string]string) {
	r.custom = aliases
}

// Resolve expands a path that may start with an alias (e.g. "@config/nvim")
// into an absolute path. A leading "~" or "~/" is expanded to the home
// directory via toolkit.ExpandPath. Paths without aliases are treated as
// relative to the home directory.
func (r *AliasResolver) Resolve(path string) (string, error) {
	path = filepath.ToSlash(path)

	if strings.HasPrefix(path, "~") {
		return toolkit.ExpandPath(r.env, path)
	}

	if !strings.HasPrefix(path, "@") {
		return filepath.Join(r.home, filepath.FromSlash(path)), nil
	}

	alias, rest := splitAlias(path)

	// Check custom aliases first (they may chain to built-in aliases).
	if r.custom != nil {
		if target, ok := r.custom[alias]; ok {
			expanded := target
			if rest != "" {
				expanded = target + "/" + rest
			}
			return r.Resolve(expanded)
		}
	}

	base, err := r.resolveBuiltin(alias)
	if err != nil {
		return "", err
	}

	if rest != "" {
		return filepath.Join(base, filepath.FromSlash(rest)), nil
	}
	return base, nil
}

// ResolveAlias returns the base directory for a single alias without any
// sub-path appended.
func (r *AliasResolver) ResolveAlias(alias string) (string, error) {
	if r.custom != nil {
		if target, ok := r.custom[alias]; ok {
			return r.Resolve(target)
		}
	}
	return r.resolveBuiltin(alias)
}

func (r *AliasResolver) resolveBuiltin(alias string) (string, error) {
	switch alias {
	case AliasHome:
		return r.home, nil
	case AliasConfig:
		return r.resolveConfig(), nil
	case AliasData:
		return r.resolveData(), nil
	case AliasCache:
		return r.resolveCache(), nil
	case AliasState:
		return r.resolveState(), nil
	case AliasBin:
		return r.resolveBin(), nil
	case AliasXDGConfig:
		return r.resolveXDGConfig(), nil
	case AliasXDGData:
		return r.resolveXDGData(), nil
	case AliasXDGCache:
		return r.resolveXDGCache(), nil
	case AliasXDGState:
		return r.resolveXDGState(), nil
	case AliasAppleConfig:
		return r.resolveAppleConfig()
	case AliasAppleData:
		return r.resolveAppleData()
	case AliasAppleCache:
		return r.resolveAppleCache()
	case AliasAppleLogs:
		return r.resolveAppleLogs()
	case AliasAppleLaunchAgents:
		return r.resolveAppleLaunchAgents()
	default:
		return "", fmt.Errorf("unknown alias: %s", alias)
	}
}

// resolveXDGConfig returns the XDG config home regardless of OS. Honors
// XDG_CONFIG_HOME if set, otherwise defaults to ~/.config (even on Windows).
func (r *AliasResolver) resolveXDGConfig() string {
	if v := r.env.Get("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	return filepath.Join(r.home, ".config")
}

// resolveXDGData returns the XDG data home regardless of OS. Honors
// XDG_DATA_HOME if set, otherwise defaults to ~/.local/share.
func (r *AliasResolver) resolveXDGData() string {
	if v := r.env.Get("XDG_DATA_HOME"); v != "" {
		return v
	}
	return filepath.Join(r.home, ".local", "share")
}

// resolveXDGCache returns the XDG cache home regardless of OS. Honors
// XDG_CACHE_HOME if set, otherwise defaults to ~/.cache.
func (r *AliasResolver) resolveXDGCache() string {
	if v := r.env.Get("XDG_CACHE_HOME"); v != "" {
		return v
	}
	return filepath.Join(r.home, ".cache")
}

// resolveXDGState returns the XDG state home regardless of OS. Honors
// XDG_STATE_HOME if set, otherwise defaults to ~/.local/state.
func (r *AliasResolver) resolveXDGState() string {
	if v := r.env.Get("XDG_STATE_HOME"); v != "" {
		return v
	}
	return filepath.Join(r.home, ".local", "state")
}

func (r *AliasResolver) resolveConfig() string {
	if r.platform.OS == "windows" {
		if v := r.env.Get("APPDATA"); v != "" {
			return v
		}
		return filepath.Join(r.home, "AppData", "Roaming")
	}
	return r.resolveXDGConfig()
}

func (r *AliasResolver) resolveData() string {
	if r.platform.OS == "windows" {
		if v := r.env.Get("LOCALAPPDATA"); v != "" {
			return v
		}
		return filepath.Join(r.home, "AppData", "Local")
	}
	return r.resolveXDGData()
}

func (r *AliasResolver) resolveCache() string {
	if r.platform.OS == "windows" {
		local := r.env.Get("LOCALAPPDATA")
		if local == "" {
			local = filepath.Join(r.home, "AppData", "Local")
		}
		return filepath.Join(local, "cache")
	}
	return r.resolveXDGCache()
}

func (r *AliasResolver) resolveState() string {
	if r.platform.OS == "windows" {
		local := r.env.Get("LOCALAPPDATA")
		if local == "" {
			local = filepath.Join(r.home, "AppData", "Local")
		}
		return filepath.Join(local, "state")
	}
	return r.resolveXDGState()
}

func (r *AliasResolver) resolveBin() string {
	if r.platform.OS == "windows" {
		local := r.env.Get("LOCALAPPDATA")
		if local == "" {
			local = filepath.Join(r.home, "AppData", "Local")
		}
		return filepath.Join(local, "bin")
	}
	return filepath.Join(r.home, ".local", "bin")
}

// appleLibrary returns the joined ~/Library/<sub> path on darwin, or an
// AliasUnavailableError on every other OS. The error wraps
// ErrAliasUnavailable so callers can detect it via errors.Is.
func (r *AliasResolver) appleLibrary(alias string, sub ...string) (string, error) {
	if r.platform.OS != "darwin" {
		return "", &AliasUnavailableError{Alias: alias, OS: r.platform.OS}
	}
	parts := append([]string{r.home, "Library"}, sub...)
	return filepath.Join(parts...), nil
}

func (r *AliasResolver) resolveAppleConfig() (string, error) {
	// Apple HIG conflates config and data under Application Support.
	return r.appleLibrary(AliasAppleConfig, "Application Support")
}

func (r *AliasResolver) resolveAppleData() (string, error) {
	// Apple HIG conflates config and data under Application Support.
	return r.appleLibrary(AliasAppleData, "Application Support")
}

func (r *AliasResolver) resolveAppleCache() (string, error) {
	return r.appleLibrary(AliasAppleCache, "Caches")
}

func (r *AliasResolver) resolveAppleLogs() (string, error) {
	return r.appleLibrary(AliasAppleLogs, "Logs")
}

func (r *AliasResolver) resolveAppleLaunchAgents() (string, error) {
	return r.appleLibrary(AliasAppleLaunchAgents, "LaunchAgents")
}

// splitAlias splits "@config/nvim/init.lua" into ("@config", "nvim/init.lua").
func splitAlias(path string) (alias, rest string) {
	idx := strings.IndexByte(path, '/')
	if idx == -1 {
		return path, ""
	}
	return path[:idx], path[idx+1:]
}
