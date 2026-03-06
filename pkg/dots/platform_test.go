package dots_test

import (
	"runtime"
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

func TestDetectPlatform(t *testing.T) {
	p := dots.DetectPlatform()
	require.Equal(t, runtime.GOOS, p.OS)
	require.Equal(t, runtime.GOARCH, p.Arch)
	require.Equal(t, runtime.GOOS+"-"+runtime.GOARCH, p.String())
}

func testEnv(vals map[string]string) func(string) string {
	return func(key string) string {
		return vals[key]
	}
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
