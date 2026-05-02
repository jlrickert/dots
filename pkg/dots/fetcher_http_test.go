package dots

import "testing"

// TestCompoundExt locks in the regression where signed-release URLs with
// query strings or fragments leaked into the cache filename. compoundExt
// must scan only the URL path component so the cache name stays free of
// characters that are invalid on NTFS (?, #) and so the canonical
// compound suffixes (.tar.gz, .tar.xz) are preserved.
func TestCompoundExt(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want string
	}{
		{"clean compound", "https://example.com/foo.tar.gz", ".tar.gz"},
		{"capitalized", "https://example.com/FOO.TAR.GZ", ".tar.gz"},
		{"query string", "https://example.com/foo.tar.gz?token=abc", ".tar.gz"},
		{"fragment", "https://example.com/foo.tar.gz#frag", ".tar.gz"},
		{"single ext", "https://example.com/foo.zip", ".zip"},
		{"no ext", "https://example.com/foo", ""},
		{"tar.xz with query", "https://example.com/foo.tar.xz?signature=xxx", ".tar.xz"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compoundExt(FetchRequest{URL: tc.url})
			if got != tc.want {
				t.Fatalf("compoundExt(%q) = %q, want %q", tc.url, got, tc.want)
			}
		})
	}
}
