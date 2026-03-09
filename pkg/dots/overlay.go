package dots

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MergeStrategy defines how overlay files are combined with base files.
type MergeStrategy string

const (
	MergeAppend  MergeStrategy = "append"
	MergePrepend MergeStrategy = "prepend"
	MergeReplace MergeStrategy = "replace"
	MergeMerge   MergeStrategy = "merge"
)

// OverlayLayer represents one layer in an overlay stack.
type OverlayLayer struct {
	Package  string // tap/package ref
	Priority int    // 0-99, higher wins
	Strategy MergeStrategy
	Dir      string // absolute path to package dir
}

// MergeFiles merges overlay layers for a single file.
// base is the content from the base package.
// layers is sorted by priority (ascending).
func MergeFiles(base []byte, layers []OverlayLayer, filename string) ([]byte, error) {
	result := base

	// Sort by priority ascending (lowest first, highest last = most specific)
	sort.Slice(layers, func(i, j int) bool {
		return layers[i].Priority < layers[j].Priority
	})

	for _, layer := range layers {
		overlayPath := filepath.Join(layer.Dir, filename)
		overlayContent, err := os.ReadFile(overlayPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read overlay file %s: %w", overlayPath, err)
		}

		strategy := layer.Strategy
		if strategy == "" {
			strategy = MergeAppend
		}

		result = applyMergeStrategy(result, overlayContent, strategy, layer.Package)
	}

	return result, nil
}

func applyMergeStrategy(base, overlay []byte, strategy MergeStrategy, origin string) []byte {
	marker := fmt.Sprintf("# --- overlay: %s ---", origin)

	switch strategy {
	case MergeReplace:
		return overlay
	case MergePrepend:
		var b strings.Builder
		b.WriteString(marker + "\n")
		b.Write(overlay)
		b.WriteString("\n" + marker + " end\n")
		b.Write(base)
		return []byte(b.String())
	case MergeMerge:
		// Simple line-level merge: base lines first, then unique overlay lines
		baseLines := strings.Split(string(base), "\n")
		overlayLines := strings.Split(string(overlay), "\n")
		seen := make(map[string]struct{})
		for _, l := range baseLines {
			seen[l] = struct{}{}
		}
		var merged []string
		merged = append(merged, baseLines...)
		for _, l := range overlayLines {
			if _, ok := seen[l]; !ok {
				merged = append(merged, l)
			}
		}
		return []byte(strings.Join(merged, "\n"))
	default: // append
		var b strings.Builder
		b.Write(base)
		b.WriteString("\n" + marker + "\n")
		b.Write(overlay)
		b.WriteString("\n" + marker + " end\n")
		return []byte(b.String())
	}
}

// WriteMergedFile writes merged output to the merged directory.
func WriteMergedFile(mergedDir, tap, pkg, filename string, data []byte) error {
	path := filepath.Join(mergedDir, tap, pkg, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
