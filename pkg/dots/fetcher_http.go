package dots

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/jlrickert/cli-toolkit/toolkit"
)

// DefaultHTTPFetchTimeout caps a single artifact fetch when the caller's
// context has no deadline. Releases that don't transfer in 10 minutes are
// almost certainly stuck on a stalled connection rather than a slow link.
const DefaultHTTPFetchTimeout = 10 * time.Minute

// HTTPFetcher fetches artifacts over net/http and persists verified bytes
// into a content-addressed cache directory. Filesystem I/O routes through
// toolkit.Runtime so that the test runtime's jailed FS captures cache
// reads/writes. cli-toolkit's Runtime does not expose an HTTP seam at
// v1.5.0, so we use net/http directly; if a seam is added upstream the
// construction site is the only thing that needs to change.
type HTTPFetcher struct {
	rt       *toolkit.Runtime
	client   *http.Client
	cacheDir string
}

// NewHTTPFetcher constructs a fetcher that caches verified bytes under
// cacheDir. Pass a nil client to use http.DefaultClient. The runtime is
// required so cache I/O is sandboxable in tests.
func NewHTTPFetcher(rt *toolkit.Runtime, client *http.Client, cacheDir string) *HTTPFetcher {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPFetcher{rt: rt, client: client, cacheDir: cacheDir}
}

// Fetch implements Fetcher. The flow is: cache lookup → conditional GET →
// hash verify → cache write → return bytes. A cache hit still re-verifies
// the digest so a corrupted cache entry surfaces as ErrChecksumMismatch
// instead of being trusted.
func (f *HTTPFetcher) Fetch(ctx context.Context, req FetchRequest) (*FetchResult, error) {
	if req.URL == "" {
		return nil, fmt.Errorf("%w: artifact url is required", ErrInvalid)
	}
	if req.Sha256 == "" {
		return nil, fmt.Errorf("%w: artifact sha256 is required", ErrInvalid)
	}

	if data, path, ok := f.cacheHit(req); ok {
		return &FetchResult{Path: path, Bytes: data}, nil
	}

	ctx, cancel := withDefaultTimeout(ctx, DefaultHTTPFetchTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("artifact request: %w", err)
	}

	resp, err := f.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("artifact fetch %s: %w", req.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("artifact fetch %s: http %d", req.URL, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("artifact read %s: %w", req.URL, err)
	}

	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != req.Sha256 {
		return nil, fmt.Errorf("%w: %s: want %s, got %s", ErrChecksumMismatch, req.URL, req.Sha256, got)
	}

	path, err := f.writeCache(req, data)
	if err != nil {
		// Cache failure is not fatal — the verified bytes are still good. Log
		// once at WARN so a read-only home or full disk is visible without
		// failing the install. Subsequent installs will simply re-download.
		if f.rt != nil {
			f.rt.Logger().Warn("artifact cache write failed",
				"url", req.URL,
				"sha256", req.Sha256,
				"err", err)
		}
		path = ""
	}
	return &FetchResult{Path: path, Bytes: data}, nil
}

// cacheHit reads a previously stored artifact and re-verifies it. A miss or a
// digest mismatch is treated as cache-cold so the caller refetches.
func (f *HTTPFetcher) cacheHit(req FetchRequest) ([]byte, string, bool) {
	if f.cacheDir == "" || f.rt == nil {
		return nil, "", false
	}
	path := f.cachePath(req)
	data, err := f.rt.ReadFile(path)
	if err != nil {
		return nil, "", false
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != req.Sha256 {
		return nil, "", false
	}
	return data, path, true
}

// writeCache persists verified bytes atomically. The runtime's
// AtomicWriteFile already implements the temp-file-and-rename dance, so the
// canonical content-addressed name never appears with a partial payload.
func (f *HTTPFetcher) writeCache(req FetchRequest, data []byte) (string, error) {
	if f.cacheDir == "" || f.rt == nil {
		return "", nil
	}
	if err := f.rt.Mkdir(f.cacheDir, 0o755, true); err != nil {
		return "", err
	}
	final := f.cachePath(req)
	if err := f.rt.AtomicWriteFile(final, data, 0o644); err != nil {
		return "", err
	}
	return final, nil
}

// cachePath builds the content-addressed cache filename. For compound
// archive extensions (.tar.gz, .tar.xz) the canonical form is preserved
// so cached artifacts inspect cleanly with the standard archive tooling.
// Single-suffix extensions (.zip, .tar, raw) fall through to filepath.Ext.
func (f *HTTPFetcher) cachePath(req FetchRequest) string {
	ext := compoundExt(req)
	name := req.Sha256
	if ext != "" {
		name = req.Sha256 + ext
	}
	return filepath.Join(f.cacheDir, name)
}

// compoundExt scans the URL path for known compound archive suffixes
// before falling back to filepath.Ext, which would otherwise collapse
// .tar.gz to .gz. The URL is parsed with net/url so query strings and
// fragments (e.g. signed-release tokens) don't bleed into the cache
// filename — leaving them in would produce names like
// "<sha>.gz?token=abc" and embed characters that are invalid on NTFS.
// If parsing fails or yields an empty path we fall back to the raw URL
// so callers still get a best-effort extension. Lower-casing handles
// upstream URLs that capitalize the archive extension.
func compoundExt(req FetchRequest) string {
	path := req.URL
	if u, err := url.Parse(req.URL); err == nil && u.Path != "" {
		path = u.Path
	}
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"):
		return ".tar.gz"
	case strings.HasSuffix(lower, ".tar.xz"):
		return ".tar.xz"
	}
	return filepath.Ext(path)
}

// withDefaultTimeout wraps ctx with a deadline only if one is not already set.
// Callers that need a different bound supply their own context.
func withDefaultTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

var _ Fetcher = (*HTTPFetcher)(nil)
