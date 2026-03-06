package dots_test

import (
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

func TestDeepMerge_Maps(t *testing.T) {
	dst := map[string]any{
		"links": map[string]any{
			"init.lua": "@config/nvim/init.lua",
			"lua/":     "@config/nvim/lua/",
		},
		"hooks": map[string]any{
			"post_install": "scripts/install.sh",
		},
	}
	src := map[string]any{
		"links": map[string]any{
			"clipboard.lua": "@config/nvim/lua/clipboard-mac.lua",
		},
		"hooks": map[string]any{
			"post_install": "scripts/install-mac.sh",
		},
	}

	result := dots.DeepMerge(dst, src)

	links := result["links"].(map[string]any)
	require.Equal(t, "@config/nvim/init.lua", links["init.lua"])
	require.Equal(t, "@config/nvim/lua/", links["lua/"])
	require.Equal(t, "@config/nvim/lua/clipboard-mac.lua", links["clipboard.lua"])

	hooks := result["hooks"].(map[string]any)
	require.Equal(t, "scripts/install-mac.sh", hooks["post_install"])
}

func TestDeepMerge_Slices(t *testing.T) {
	dst := map[string]any{
		"tags": []string{"editor", "neovim"},
	}
	src := map[string]any{
		"tags": []string{"neovim", "mac"},
	}

	result := dots.DeepMerge(dst, src)
	tags := result["tags"].([]string)
	require.Equal(t, []string{"editor", "neovim", "mac"}, tags)
}

func TestDeepMerge_Scalars(t *testing.T) {
	dst := map[string]any{
		"link_strategy": "symlink",
		"name":          "nvim",
	}
	src := map[string]any{
		"link_strategy": "copy",
	}

	result := dots.DeepMerge(dst, src)
	require.Equal(t, "copy", result["link_strategy"])
	require.Equal(t, "nvim", result["name"])
}

func TestDeepMerge_NewKeys(t *testing.T) {
	dst := map[string]any{"a": "1"}
	src := map[string]any{"b": "2"}

	result := dots.DeepMerge(dst, src)
	require.Equal(t, "1", result["a"])
	require.Equal(t, "2", result["b"])
}

func TestDeepMerge_NilDst(t *testing.T) {
	src := map[string]any{"key": "value"}
	result := dots.DeepMerge(nil, src)
	require.Equal(t, "value", result["key"])
}

func TestResolvePlatformCascade(t *testing.T) {
	base := map[string]any{
		"links": map[string]any{
			"init.lua": "@config/nvim/init.lua",
			"lua/":     "@config/nvim/lua/",
		},
		"hooks": map[string]any{
			"post_install": "scripts/install.sh",
		},
	}
	darwin := map[string]any{
		"links": map[string]any{
			"clipboard.lua": "@config/nvim/lua/clipboard-mac.lua",
		},
		"hooks": map[string]any{
			"post_install": "scripts/install-mac.sh",
		},
	}
	darwinArm64 := map[string]any{
		"links": map[string]any{
			"bin/nvim-arm": "@bin/nvim-silicon",
		},
	}

	result := dots.ResolvePlatformCascade(base, darwin, darwinArm64)

	links := result["links"].(map[string]any)
	require.Equal(t, "@config/nvim/init.lua", links["init.lua"])
	require.Equal(t, "@config/nvim/lua/", links["lua/"])
	require.Equal(t, "@config/nvim/lua/clipboard-mac.lua", links["clipboard.lua"])
	require.Equal(t, "@bin/nvim-silicon", links["bin/nvim-arm"])

	hooks := result["hooks"].(map[string]any)
	require.Equal(t, "scripts/install-mac.sh", hooks["post_install"])
}

func TestResolvePlatformCascade_NilSections(t *testing.T) {
	base := map[string]any{
		"links": map[string]any{
			"init.lua": "@config/nvim/init.lua",
		},
	}

	result := dots.ResolvePlatformCascade(base, nil, nil)

	links := result["links"].(map[string]any)
	require.Equal(t, "@config/nvim/init.lua", links["init.lua"])
}

func TestResolvePlatformCascade_ArchOverridesOS(t *testing.T) {
	base := map[string]any{
		"link_strategy": "symlink",
	}
	osSection := map[string]any{
		"link_strategy": "copy",
	}
	archSection := map[string]any{
		"link_strategy": "hardlink",
	}

	result := dots.ResolvePlatformCascade(base, osSection, archSection)
	require.Equal(t, "hardlink", result["link_strategy"])
}
