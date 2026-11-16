package official

import (
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
		})
	}
}
