package official

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
)

func TestGoHTTPClients_RejectHTTPDowngradeRedirect(t *testing.T) {
	clients := []struct {
		name   string
		client *http.Client
	}{
		{"defaultGoCheckClient", defaultGoCheckClient},
		{"defaultGoDownloadClient", defaultGoDownloadClient},
	}

	httpReq := &http.Request{URL: &url.URL{Scheme: "http", Host: "example.com"}}
	httpsReq := &http.Request{URL: &url.URL{Scheme: "https", Host: "example.com"}}

	for _, tc := range clients {
		t.Run(tc.name, func(t *testing.T) {
			if tc.client.CheckRedirect == nil {
				t.Fatal("CheckRedirect = nil, want HTTPS-only policy")
			}
			if err := tc.client.CheckRedirect(httpReq, nil); err == nil {
				t.Error("CheckRedirect(http) = nil, want error")
			}
			if err := tc.client.CheckRedirect(httpsReq, nil); err != nil {
				t.Errorf("CheckRedirect(https) = %v, want nil", err)
			}
			tenHops := make([]*http.Request, 10)
			if err := tc.client.CheckRedirect(httpsReq, tenHops); err == nil {
				t.Error("CheckRedirect with 10 hops = nil, want redirect limit error")
			}
		})
	}
}

type stubRoundTripper func(*http.Request) (*http.Response, error)

func (f stubRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type chunkedBody struct {
	chunks []string
	idx    int
	err    error
}

func (b *chunkedBody) Read(p []byte) (int, error) {
	if b.idx < len(b.chunks) {
		n := copy(p, b.chunks[b.idx])
		b.idx++
		return n, nil
	}
	if b.err != nil {
		return 0, b.err
	}
	return 0, io.EOF
}

func (b *chunkedBody) Close() error { return nil }

func TestFetchGoDevVersion_ReadsAcrossChunksAndPropagatesError(t *testing.T) {
	origTransport := defaultGoCheckClient.Transport
	defer func() { defaultGoCheckClient.Transport = origTransport }()

	t.Run("reads full version across multiple Read calls", func(t *testing.T) {
		defaultGoCheckClient.Transport = stubRoundTripper(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       &chunkedBody{chunks: []string{"go1.", "27.4\n", "time 2026-01-01T00:00:00Z\n"}},
			}, nil
		})
		got, err := fetchGoDevVersion(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "go1.27.4" {
			t.Errorf("got %q, want go1.27.4", got)
		}
	})

	t.Run("propagates body read error", func(t *testing.T) {
		readErr := errors.New("connection reset")
		defaultGoCheckClient.Transport = stubRoundTripper(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       &chunkedBody{chunks: []string{"go1."}, err: readErr},
			}, nil
		})
		_, err := fetchGoDevVersion(context.Background())
		if !errors.Is(err, readErr) {
			t.Fatalf("err = %v, want %v", err, readErr)
		}
	})
}

func TestFetchGoRelease_RejectsUnsafeFilename(t *testing.T) {
	origTransport := defaultGoCheckClient.Transport
	defer func() { defaultGoCheckClient.Transport = origTransport }()

	defaultGoCheckClient.Transport = stubRoundTripper(func(*http.Request) (*http.Response, error) {
		body := `[{"version":"go1.27.4","stable":true,"files":[{"filename":"../../evil.tar.gz","os":"linux","arch":"amd64","kind":"archive","sha256":"abc"}]}]`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(&chunkedBody{chunks: []string{body}}),
		}, nil
	})
	if _, err := fetchGoRelease(context.Background(), "linux", "amd64"); err == nil {
		t.Fatal("fetchGoRelease accepted traversal filename, want error")
	}
}
