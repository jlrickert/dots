package dots

// DeepMerge merges src into dst. Maps are recursively merged, scalars are
// replaced, and slices are concatenated with duplicates removed. dst is
// modified in place and returned.
func DeepMerge(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = make(map[string]any)
	}
	for k, sv := range src {
		dv, exists := dst[k]
		if !exists {
			dst[k] = deepCopy(sv)
			continue
		}

		dstMap, dstIsMap := dv.(map[string]any)
		srcMap, srcIsMap := sv.(map[string]any)
		if dstIsMap && srcIsMap {
			dst[k] = DeepMerge(dstMap, srcMap)
			continue
		}

		dstSlice, dstIsSlice := toStringSlice(dv)
		srcSlice, srcIsSlice := toStringSlice(sv)
		if dstIsSlice && srcIsSlice {
			dst[k] = dedup(append(dstSlice, srcSlice...))
			continue
		}

		// Scalar replacement: more specific wins.
		dst[k] = deepCopy(sv)
	}
	return dst
}

// ResolvePlatformCascade merges platform sections in order of specificity:
// base → OS-only → OS-arch. Returns the effective configuration.
func ResolvePlatformCascade(base, osSection, archSection map[string]any) map[string]any {
	result := make(map[string]any)
	DeepMerge(result, base)
	if osSection != nil {
		DeepMerge(result, osSection)
	}
	if archSection != nil {
		DeepMerge(result, archSection)
	}
	return result
}

func deepCopy(v any) any {
	switch val := v.(type) {
	case map[string]any:
		cp := make(map[string]any, len(val))
		for k, v := range val {
			cp[k] = deepCopy(v)
		}
		return cp
	case []any:
		cp := make([]any, len(val))
		for i, v := range val {
			cp[i] = deepCopy(v)
		}
		return cp
	case []string:
		cp := make([]string, len(val))
		copy(cp, val)
		return cp
	default:
		return v
	}
}

func toStringSlice(v any) ([]string, bool) {
	switch val := v.(type) {
	case []string:
		return val, true
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			result = append(result, s)
		}
		return result, true
	default:
		return nil, false
	}
}

func dedup(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
