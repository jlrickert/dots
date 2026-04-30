package dots_test

import (
	"errors"
	"runtime"
	"testing"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

func TestDetectPlatform(t *testing.T) {
	p := dots.DetectPlatform()
	require.Equal(t, runtime.GOOS, p.OS)
	require.Equal(t, runtime.GOARCH, p.Arch)
	require.Equal(t, runtime.GOOS+"-"+runtime.GOARCH, p.String())
}

// mapEnv is a minimal toolkit.Env backed by a map. Only Get and GetHome
// carry meaningful behavior; other methods exist to satisfy the interface.
type mapEnv map[string]string

func (m mapEnv) Name() string              { return "mapEnv" }
func (m mapEnv) GetJail() string           { return "" }
func (m mapEnv) SetJail(string) error      { return nil }
func (m mapEnv) Get(key string) string     { return m[key] }
func (m mapEnv) Set(key, val string) error { m[key] = val; return nil }
func (m mapEnv) Has(key string) bool       { _, ok := m[key]; return ok }
func (m mapEnv) Environ() []string         { return nil }
func (m mapEnv) Unset(key string)          { delete(m, key) }
func (m mapEnv) GetHome() (string, error) {
	if v := m["USERPROFILE"]; v != "" {
		return v, nil
	}
	return m["HOME"], nil
}
func (m mapEnv) SetHome(h string) error   { m["HOME"] = h; return nil }
func (m mapEnv) GetUser() (string, error) { return m["USER"], nil }
func (m mapEnv) SetUser(u string) error   { m["USER"] = u; return nil }
func (m mapEnv) Getwd() (string, error)   { return m["PWD"], nil }
func (m mapEnv) Setwd(d string) error     { m["PWD"] = d; return nil }
func (m mapEnv) GetTempDir() string       { return m["TMPDIR"] }

func testEnv(vals map[string]string) toolkit.Env {
	return mapEnv(vals)
}

func TestAliasResolver_Unix(t *testing.T) {
	p := dots.Platform{OS: "linux", Arch: "amd64"}
	env := testEnv(map[string]string{"HOME": "/home/user"})
	r := dots.NewAliasResolver(p, env)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"home", "@home", "/home/user"},
		{"config default", "@config", "/home/user/.config"},
		{"data default", "@data", "/home/user/.local/share"},
		{"cache default", "@cache", "/home/user/.cache"},
		{"state default", "@state", "/home/user/.local/state"},
		{"bin default", "@bin", "/home/user/.local/bin"},
		{"config subpath", "@config/nvim/init.lua", "/home/user/.config/nvim/init.lua"},
		{"raw path", ".gitconfig", "/home/user/.gitconfig"},
		{"tilde alone", "~", "/home/user"},
		{"tilde subpath", "~/Library/LaunchAgents/foo.plist", "/home/user/Library/LaunchAgents/foo.plist"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Resolve(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestAliasResolver_UnixXDGOverride(t *testing.T) {
	p := dots.Platform{OS: "darwin", Arch: "arm64"}
	env := testEnv(map[string]string{
		"HOME":            "/Users/me",
		"XDG_CONFIG_HOME": "/custom/config",
		"XDG_DATA_HOME":   "/custom/data",
		"XDG_CACHE_HOME":  "/custom/cache",
		"XDG_STATE_HOME":  "/custom/state",
	})
	r := dots.NewAliasResolver(p, env)

	tests := []struct {
		alias string
		want  string
	}{
		{"@config", "/custom/config"},
		{"@data", "/custom/data"},
		{"@cache", "/custom/cache"},
		{"@state", "/custom/state"},
		{"@bin", "/Users/me/.local/bin"},
	}
	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			got, err := r.ResolveAlias(tt.alias)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestAliasResolver_Windows(t *testing.T) {
	p := dots.Platform{OS: "windows", Arch: "amd64"}
	env := testEnv(map[string]string{
		"USERPROFILE":  "C:/Users/me",
		"APPDATA":      "C:/Users/me/AppData/Roaming",
		"LOCALAPPDATA": "C:/Users/me/AppData/Local",
	})
	r := dots.NewAliasResolver(p, env)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"home", "@home", "C:/Users/me"},
		{"config", "@config", "C:/Users/me/AppData/Roaming"},
		{"data", "@data", "C:/Users/me/AppData/Local"},
		{"cache", "@cache", "C:/Users/me/AppData/Local/cache"},
		{"state", "@state", "C:/Users/me/AppData/Local/state"},
		{"bin", "@bin", "C:/Users/me/AppData/Local/bin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Resolve(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestAliasResolver_CustomAliases(t *testing.T) {
	p := dots.Platform{OS: "linux", Arch: "amd64"}
	env := testEnv(map[string]string{"HOME": "/home/user"})
	r := dots.NewAliasResolver(p, env)
	r.SetCustomAliases(map[string]string{
		"@dots":    "@config/dots",
		"@nvim":    "@config/nvim",
		"@scripts": "@home/scripts",
	})

	tests := []struct {
		input string
		want  string
	}{
		{"@dots", "/home/user/.config/dots"},
		{"@dots/config.yaml", "/home/user/.config/dots/config.yaml"},
		{"@nvim/init.lua", "/home/user/.config/nvim/init.lua"},
		{"@scripts/backup.sh", "/home/user/scripts/backup.sh"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := r.Resolve(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestAliasResolver_UnknownAlias(t *testing.T) {
	p := dots.Platform{OS: "linux", Arch: "amd64"}
	env := testEnv(map[string]string{"HOME": "/home/user"})
	r := dots.NewAliasResolver(p, env)

	_, err := r.Resolve("@unknown/foo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown alias")
}

// TestAliasResolver_XDGFamily_Linux confirms that the @xdg-* aliases resolve
// to the standard XDG defaults under a controlled HOME on Linux.
func TestAliasResolver_XDGFamily_Linux(t *testing.T) {
	p := dots.Platform{OS: "linux", Arch: "amd64"}
	env := testEnv(map[string]string{"HOME": "/home/user"})
	r := dots.NewAliasResolver(p, env)

	tests := []struct {
		input string
		want  string
	}{
		{"@xdg-config", "/home/user/.config"},
		{"@xdg-data", "/home/user/.local/share"},
		{"@xdg-cache", "/home/user/.cache"},
		{"@xdg-state", "/home/user/.local/state"},
		{"@xdg-config/nvim/init.lua", "/home/user/.config/nvim/init.lua"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := r.Resolve(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestAliasResolver_XDGFamily_Darwin confirms the same XDG defaults apply on
// darwin even though the platform is not Linux.
func TestAliasResolver_XDGFamily_Darwin(t *testing.T) {
	p := dots.Platform{OS: "darwin", Arch: "arm64"}
	env := testEnv(map[string]string{"HOME": "/Users/me"})
	r := dots.NewAliasResolver(p, env)

	tests := []struct {
		input string
		want  string
	}{
		{"@xdg-config", "/Users/me/.config"},
		{"@xdg-data", "/Users/me/.local/share"},
		{"@xdg-cache", "/Users/me/.cache"},
		{"@xdg-state", "/Users/me/.local/state"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := r.Resolve(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestAliasResolver_XDGFamily_Windows is the critical case: even with APPDATA
// and LOCALAPPDATA set, @xdg-config must resolve to <HOME>/.config (not
// %APPDATA%). XDG_CONFIG_HOME is honored when set.
func TestAliasResolver_XDGFamily_Windows(t *testing.T) {
	p := dots.Platform{OS: "windows", Arch: "amd64"}

	t.Run("defaults_ignore_AppData", func(t *testing.T) {
		env := testEnv(map[string]string{
			"USERPROFILE":  "C:/Users/me",
			"APPDATA":      "C:/Users/me/AppData/Roaming",
			"LOCALAPPDATA": "C:/Users/me/AppData/Local",
		})
		r := dots.NewAliasResolver(p, env)

		tests := []struct {
			input string
			want  string
		}{
			{"@xdg-config", "C:/Users/me/.config"},
			{"@xdg-data", "C:/Users/me/.local/share"},
			{"@xdg-cache", "C:/Users/me/.cache"},
			{"@xdg-state", "C:/Users/me/.local/state"},
		}
		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				got, err := r.Resolve(tt.input)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("XDG_CONFIG_HOME_honored", func(t *testing.T) {
		env := testEnv(map[string]string{
			"USERPROFILE":     "C:/Users/me",
			"APPDATA":         "C:/Users/me/AppData/Roaming",
			"LOCALAPPDATA":    "C:/Users/me/AppData/Local",
			"XDG_CONFIG_HOME": "C:/custom/xdg-config",
		})
		r := dots.NewAliasResolver(p, env)

		got, err := r.Resolve("@xdg-config/nvim")
		require.NoError(t, err)
		require.Equal(t, "C:/custom/xdg-config/nvim", got)
	})
}

// TestAliasResolver_XDGFamily_EnvOverrides verifies XDG_*_HOME env vars are
// honored on each platform.
func TestAliasResolver_XDGFamily_EnvOverrides(t *testing.T) {
	platforms := []struct {
		name string
		p    dots.Platform
		home string
	}{
		{"linux", dots.Platform{OS: "linux", Arch: "amd64"}, "/home/user"},
		{"darwin", dots.Platform{OS: "darwin", Arch: "arm64"}, "/Users/me"},
		{"windows", dots.Platform{OS: "windows", Arch: "amd64"}, "C:/Users/me"},
	}
	for _, plat := range platforms {
		t.Run(plat.name, func(t *testing.T) {
			env := testEnv(map[string]string{
				"HOME":            plat.home,
				"USERPROFILE":     plat.home,
				"APPDATA":         "ignored/appdata",
				"LOCALAPPDATA":    "ignored/localappdata",
				"XDG_CONFIG_HOME": "/custom/cfg",
				"XDG_DATA_HOME":   "/custom/dat",
				"XDG_CACHE_HOME":  "/custom/cch",
				"XDG_STATE_HOME":  "/custom/stt",
			})
			r := dots.NewAliasResolver(plat.p, env)

			tests := []struct {
				alias string
				want  string
			}{
				{"@xdg-config", "/custom/cfg"},
				{"@xdg-data", "/custom/dat"},
				{"@xdg-cache", "/custom/cch"},
				{"@xdg-state", "/custom/stt"},
			}
			for _, tt := range tests {
				got, err := r.ResolveAlias(tt.alias)
				require.NoError(t, err, tt.alias)
				require.Equal(t, tt.want, got, tt.alias)
			}
		})
	}
}

// TestAliasResolver_AppleFamily_Darwin verifies each @apple-* alias resolves
// to the correct ~/Library/... subdirectory on darwin and that subpath joins
// work correctly.
func TestAliasResolver_AppleFamily_Darwin(t *testing.T) {
	p := dots.Platform{OS: "darwin", Arch: "arm64"}
	env := testEnv(map[string]string{"HOME": "/Users/me"})
	r := dots.NewAliasResolver(p, env)

	tests := []struct {
		input string
		want  string
	}{
		{"@apple-config", "/Users/me/Library/Application Support"},
		{"@apple-data", "/Users/me/Library/Application Support"},
		{"@apple-cache", "/Users/me/Library/Caches"},
		{"@apple-logs", "/Users/me/Library/Logs"},
		{"@apple-launchagents", "/Users/me/Library/LaunchAgents"},
		{
			"@apple-launchagents/com.user.foo.plist",
			"/Users/me/Library/LaunchAgents/com.user.foo.plist",
		},
		{
			"@apple-config/dots/config.yaml",
			"/Users/me/Library/Application Support/dots/config.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := r.Resolve(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestAliasResolver_AppleFamily_NonDarwin verifies that on linux and windows,
// every @apple-* alias returns an AliasUnavailableError that satisfies
// errors.Is(err, ErrAliasUnavailable) and includes the alias name in the
// message.
func TestAliasResolver_AppleFamily_NonDarwin(t *testing.T) {
	platforms := []struct {
		name string
		p    dots.Platform
		env  map[string]string
	}{
		{"linux", dots.Platform{OS: "linux", Arch: "amd64"}, map[string]string{"HOME": "/home/user"}},
		{"windows", dots.Platform{OS: "windows", Arch: "amd64"}, map[string]string{
			"USERPROFILE":  "C:/Users/me",
			"APPDATA":      "C:/Users/me/AppData/Roaming",
			"LOCALAPPDATA": "C:/Users/me/AppData/Local",
		}},
	}
	aliases := []string{
		"@apple-config",
		"@apple-data",
		"@apple-cache",
		"@apple-logs",
		"@apple-launchagents",
	}
	for _, plat := range platforms {
		t.Run(plat.name, func(t *testing.T) {
			r := dots.NewAliasResolver(plat.p, testEnv(plat.env))
			for _, alias := range aliases {
				t.Run(alias, func(t *testing.T) {
					_, err := r.Resolve(alias + "/foo.txt")
					require.Error(t, err)
					require.True(t, errors.Is(err, dots.ErrAliasUnavailable),
						"errors.Is(err, ErrAliasUnavailable) should be true; err=%v", err)

					var aerr *dots.AliasUnavailableError
					require.True(t, errors.As(err, &aerr),
						"errors.As should match AliasUnavailableError; err=%v", err)
					require.Equal(t, alias, aerr.Alias)
					require.Equal(t, plat.p.OS, aerr.OS)
					require.Contains(t, err.Error(), alias)
				})
			}
		})
	}
}

// TestAliasResolver_CustomAliasChainsAppleFamily verifies a custom alias can
// chain through an Apple-family builtin: succeeds on darwin, surfaces the
// typed error on Linux.
func TestAliasResolver_CustomAliasChainsAppleFamily(t *testing.T) {
	t.Run("darwin", func(t *testing.T) {
		p := dots.Platform{OS: "darwin", Arch: "arm64"}
		env := testEnv(map[string]string{"HOME": "/Users/me"})
		r := dots.NewAliasResolver(p, env)
		r.SetCustomAliases(map[string]string{
			"@launchd": "@apple-launchagents",
		})

		got, err := r.Resolve("@launchd/com.user.foo.plist")
		require.NoError(t, err)
		require.Equal(t, "/Users/me/Library/LaunchAgents/com.user.foo.plist", got)
	})

	t.Run("linux", func(t *testing.T) {
		p := dots.Platform{OS: "linux", Arch: "amd64"}
		env := testEnv(map[string]string{"HOME": "/home/user"})
		r := dots.NewAliasResolver(p, env)
		r.SetCustomAliases(map[string]string{
			"@launchd": "@apple-launchagents",
		})

		_, err := r.Resolve("@launchd/com.user.foo.plist")
		require.Error(t, err)
		require.True(t, errors.Is(err, dots.ErrAliasUnavailable))
	})
}

// TestBuiltinAliasesRoundTrip iterates every entry in BuiltinAliases and
// asserts each alias either resolves successfully (via ResolveAlias) or
// returns ErrAliasUnavailable. This catches missing wiring in
// resolveBuiltin's switch.
func TestBuiltinAliasesRoundTrip(t *testing.T) {
	platforms := []struct {
		name string
		p    dots.Platform
		env  map[string]string
	}{
		{"darwin", dots.Platform{OS: "darwin", Arch: "arm64"}, map[string]string{"HOME": "/Users/me"}},
		{"linux", dots.Platform{OS: "linux", Arch: "amd64"}, map[string]string{"HOME": "/home/user"}},
		{"windows", dots.Platform{OS: "windows", Arch: "amd64"}, map[string]string{
			"USERPROFILE":  "C:/Users/me",
			"APPDATA":      "C:/Users/me/AppData/Roaming",
			"LOCALAPPDATA": "C:/Users/me/AppData/Local",
		}},
	}
	for _, plat := range platforms {
		t.Run(plat.name, func(t *testing.T) {
			r := dots.NewAliasResolver(plat.p, testEnv(plat.env))
			for _, alias := range dots.BuiltinAliases {
				t.Run(alias, func(t *testing.T) {
					got, err := r.ResolveAlias(alias)
					if err != nil {
						require.True(t, errors.Is(err, dots.ErrAliasUnavailable),
							"alias %q on %s returned non-availability error: %v",
							alias, plat.p.OS, err)
						return
					}
					require.NotEmpty(t, got)
				})
			}
		})
	}
}

// TestSplitAlias_HyphenatedNames is a regression test confirming splitAlias
// handles hyphenated alias names like @xdg-config/foo correctly. Since
// splitAlias is unexported, we exercise it through Resolve.
func TestSplitAlias_HyphenatedNames(t *testing.T) {
	p := dots.Platform{OS: "linux", Arch: "amd64"}
	env := testEnv(map[string]string{"HOME": "/home/user"})
	r := dots.NewAliasResolver(p, env)

	got, err := r.Resolve("@xdg-config/foo")
	require.NoError(t, err)
	require.Equal(t, "/home/user/.config/foo", got)

	// Also confirm the alias name itself (no subpath) round-trips.
	got, err = r.Resolve("@xdg-config")
	require.NoError(t, err)
	require.Equal(t, "/home/user/.config", got)
}
