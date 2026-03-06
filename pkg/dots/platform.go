package dots

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
)

// BuiltinAliases is the ordered list of built-in alias names.
var BuiltinAliases = []Alias{
	AliasHome, AliasConfig, AliasData, AliasCache, AliasState, AliasBin,
}

// AliasResolver resolves path aliases to absolute paths for a given platform.
type AliasResolver struct {
	platform Platform
	env      func(string) string
	home     string
	custom   map[string]string
}

// NewAliasResolver creates a resolver for the given platform. The env function
// is used to look up environment variables (typically os.Getenv).
func NewAliasResolver(p Platform, env func(string) string) *AliasResolver {
	home := resolveHome(p, env)
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
// into an absolute path. Paths without aliases are treated as relative to
// the home directory.
func (r *AliasResolver) Resolve(path string) (string, error) {
	path = filepath.ToSlash(path)

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
	default:
		return "", fmt.Errorf("unknown alias: %s", alias)
	}
}

func (r *AliasResolver) resolveConfig() string {
	if r.platform.OS == "windows" {
		if v := r.env("APPDATA"); v != "" {
			return v
		}
		return filepath.Join(r.home, "AppData", "Roaming")
	}
	if v := r.env("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	return filepath.Join(r.home, ".config")
}

func (r *AliasResolver) resolveData() string {
	if r.platform.OS == "windows" {
		if v := r.env("LOCALAPPDATA"); v != "" {
			return v
		}
		return filepath.Join(r.home, "AppData", "Local")
	}
	if v := r.env("XDG_DATA_HOME"); v != "" {
		return v
	}
	return filepath.Join(r.home, ".local", "share")
}

func (r *AliasResolver) resolveCache() string {
	if r.platform.OS == "windows" {
		local := r.env("LOCALAPPDATA")
		if local == "" {
			local = filepath.Join(r.home, "AppData", "Local")
		}
		return filepath.Join(local, "cache")
	}
	if v := r.env("XDG_CACHE_HOME"); v != "" {
		return v
	}
	return filepath.Join(r.home, ".cache")
}

func (r *AliasResolver) resolveState() string {
	if r.platform.OS == "windows" {
		local := r.env("LOCALAPPDATA")
		if local == "" {
			local = filepath.Join(r.home, "AppData", "Local")
		}
		return filepath.Join(local, "state")
	}
	if v := r.env("XDG_STATE_HOME"); v != "" {
		return v
	}
	return filepath.Join(r.home, ".local", "state")
}

func (r *AliasResolver) resolveBin() string {
	if r.platform.OS == "windows" {
		local := r.env("LOCALAPPDATA")
		if local == "" {
			local = filepath.Join(r.home, "AppData", "Local")
		}
		return filepath.Join(local, "bin")
	}
	return filepath.Join(r.home, ".local", "bin")
}

func resolveHome(p Platform, env func(string) string) string {
	if p.OS == "windows" {
		if v := env("USERPROFILE"); v != "" {
			return v
		}
		return `C:\Users\default`
	}
	if v := env("HOME"); v != "" {
		return v
	}
	return os.Getenv("HOME")
}

// splitAlias splits "@config/nvim/init.lua" into ("@config", "nvim/init.lua").
func splitAlias(path string) (alias, rest string) {
	idx := strings.IndexByte(path, '/')
	if idx == -1 {
		return path, ""
	}
	return path[:idx], path[idx+1:]
}
