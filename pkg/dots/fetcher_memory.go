package dots

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

// MemoryFetcher is an in-memory Fetcher used in tests. It is keyed by URL
// and round-trips bytes verbatim, but still enforces the checksum contract
// so that "corrupt fixture" cases can be exercised without touching a real
// HTTP server.
type MemoryFetcher struct {
	mu    sync.RWMutex
	store map[string][]byte
}

// NewMemoryFetcher returns an empty fetcher.
func NewMemoryFetcher() *MemoryFetcher {
	return &MemoryFetcher{store: make(map[string][]byte)}
}

// Add seeds a URL → bytes mapping. Test helpers call this directly with the
// fixture payload; production callers never touch a MemoryFetcher.
func (f *MemoryFetcher) Add(url string, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	f.store[url] = cp
}

// Fetch implements Fetcher. The sha256 enforcement happens here, not at the
// caller, so tests for ErrChecksumMismatch can use this fetcher unchanged.
func (f *MemoryFetcher) Fetch(ctx context.Context, req FetchRequest) (*FetchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.URL == "" {
		return nil, fmt.Errorf("%w: artifact url is required", ErrInvalid)
	}
	if req.Sha256 == "" {
		return nil, fmt.Errorf("%w: artifact sha256 is required", ErrInvalid)
	}

	f.mu.RLock()
	data, ok := f.store[req.URL]
	f.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotExist, req.URL)
	}

	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != req.Sha256 {
		return nil, fmt.Errorf("%w: %s: want %s, got %s", ErrChecksumMismatch, req.URL, req.Sha256, got)
	}

	cp := make([]byte, len(data))
	copy(cp, data)
	return &FetchResult{Path: req.URL, Bytes: cp}, nil
}

var _ Fetcher = (*MemoryFetcher)(nil)
