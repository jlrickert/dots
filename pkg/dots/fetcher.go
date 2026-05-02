package dots

import "context"

// FetchRequest describes a single artifact fetch. The Sha256 is mandatory —
// callers that need a checksum-less download should compute and supply one
// up front. Forcing the field at the type boundary keeps the no-trust
// invariant local to the request: the fetcher cannot be misused to surface
// unverified bytes.
type FetchRequest struct {
	URL    string
	Sha256 string
}

// FetchResult carries the verified bytes of a single artifact along with the
// content-addressed cache path. The path is informational — callers should
// not trust it as a working location for extraction. Implementations are
// free to return the bytes unchanged from a cache hit.
type FetchResult struct {
	Path  string
	Bytes []byte
}

// Fetcher retrieves artifact bytes by URL and verifies them against an
// expected sha256. Implementations MUST return ErrChecksumMismatch when the
// returned bytes do not hash to the requested digest; returning bytes
// without checking is a contract violation.
type Fetcher interface {
	Fetch(ctx context.Context, req FetchRequest) (*FetchResult, error)
}
