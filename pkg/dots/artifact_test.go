package dots_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/jlrickert/dots/pkg/dots"
	"github.com/stretchr/testify/require"
)

func sha256Hex(t *testing.T, data []byte) string {
	t.Helper()
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestFetcher_Memory_RoundTrip(t *testing.T) {
	f := dots.NewMemoryFetcher()
	payload := []byte("hello world")
	url := "https://example.test/release-1.0.0.tar.gz"
	f.Add(url, payload)

	res, err := f.Fetch(context.Background(), dots.FetchRequest{
		URL:    url,
		Sha256: sha256Hex(t, payload),
	})
	require.NoError(t, err)
	require.Equal(t, payload, res.Bytes)
	require.Equal(t, url, res.Path)
}

func TestFetcher_Memory_ChecksumMismatch(t *testing.T) {
	f := dots.NewMemoryFetcher()
	url := "https://example.test/corrupt.tar.gz"
	f.Add(url, []byte("real bytes"))

	_, err := f.Fetch(context.Background(), dots.FetchRequest{
		URL:    url,
		Sha256: sha256Hex(t, []byte("expected-something-else")),
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, dots.ErrChecksumMismatch))
}

func TestFetcher_Memory_MissingURL(t *testing.T) {
	f := dots.NewMemoryFetcher()
	_, err := f.Fetch(context.Background(), dots.FetchRequest{
		URL:    "https://example.test/missing",
		Sha256: sha256Hex(t, []byte("x")),
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, dots.ErrNotExist))
}

func TestFetcher_Memory_Validation(t *testing.T) {
	f := dots.NewMemoryFetcher()
	cases := []struct {
		name string
		req  dots.FetchRequest
	}{
		{"missing url", dots.FetchRequest{Sha256: sha256Hex(t, []byte("x"))}},
		{"missing sha256", dots.FetchRequest{URL: "https://example.test/x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.Fetch(context.Background(), tc.req)
			require.Error(t, err)
			require.True(t, errors.Is(err, dots.ErrInvalid))
		})
	}
}

func TestArtifact_FormatConstants(t *testing.T) {
	// Sanity-pin the YAML wire values; manifests refer to these by name.
	require.Equal(t, "tar.gz", string(dots.ExtractTarGz))
	require.Equal(t, "tar.xz", string(dots.ExtractTarXz))
	require.Equal(t, "zip", string(dots.ExtractZip))
	require.Equal(t, "", string(dots.ExtractNone))
}
