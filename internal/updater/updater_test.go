package updater

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fakeHTTP func(*http.Request) (*http.Response, error)

func (fake fakeHTTP) Do(request *http.Request) (*http.Response, error) {
	return fake(request)
}

func TestReleaseChecksumReadsGitHubChecksumFormat(t *testing.T) {
	t.Parallel()
	want := strings.Repeat("a", 64)
	installer := &Installer{http: fakeHTTP(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(want + "  ./bluff_v0.1.4_darwin_arm64.tar.gz\n")),
		}, nil
	})}

	got, err := installer.releaseChecksum(context.Background(), "https://example.test/checksums.txt", "bluff_v0.1.4_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatalf("releaseChecksum() error = %v", err)
	}
	if got != want {
		t.Fatalf("releaseChecksum() = %q, want %q", got, want)
	}
}

func TestValidReleaseVersion(t *testing.T) {
	t.Parallel()
	if !validReleaseVersion("v0.1.4") {
		t.Fatal("expected stable release version to be valid")
	}
	if validReleaseVersion("https://example.test") {
		t.Fatal("expected URL to be rejected as a release version")
	}
}
