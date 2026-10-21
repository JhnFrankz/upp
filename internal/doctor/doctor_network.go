package doctor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// defaultDoctorHTTPClient is reused across HTTP checks to enable connection pooling and keep-alives.
var defaultDoctorHTTPClient = &http.Client{
	Timeout: 3 * time.Second,
}

func defaultHTTPGet(ctx context.Context, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "upp")
	resp, err := defaultDoctorHTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
		_ = resp.Body.Close()
	}()
	return resp.StatusCode, nil
}

func checkNetwork(ctx context.Context, results []CheckResult, deps DoctorDeps) []CheckResult {
	targets := []struct {
		name string
		url  string
	}{
		{"api.github.com", "https://api.github.com"},
		{"go.dev", "https://go.dev"},
	}

	netResults := make([]CheckResult, len(targets))
	var wg sync.WaitGroup

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, target struct{ name, url string }) {
			defer wg.Done()

			if ctx.Err() != nil {
				netResults[idx] = CheckResult{
					Category: "Network",
					Name:     target.name,
					Status:   SeverityWarn,
					Message:  fmt.Sprintf("Check aborted: %v", ctx.Err()),
					Detail:   target.url,
				}
				return
			}

			status, err := deps.HTTPGet(ctx, target.url)
			if err != nil || status < 200 || status >= 400 {
				var msg string
				if err != nil {
					msg = fmt.Sprintf("Failed to reach %s: %v", target.url, err)
				} else {
					msg = fmt.Sprintf("Failed to reach %s (HTTP %d)", target.url, status)
				}
				netResults[idx] = CheckResult{
					Category: "Network",
					Name:     target.name,
					Status:   SeverityWarn,
					Message:  msg,
					Detail:   target.url,
					FixHint:  "Check your internet connection, DNS, or proxy settings.",
				}
			} else {
				netResults[idx] = CheckResult{
					Category: "Network",
					Name:     target.name,
					Status:   SeverityOK,
					Message:  fmt.Sprintf("Reachable (status %d)", status),
					Detail:   target.url,
				}
			}
		}(i, t)
	}

	wg.Wait()
	return append(results, netResults...)
}
