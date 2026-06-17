package sublink

import "testing"

// TestBuildFetchClient covers the per-mode client construction and validation.
func TestBuildFetchClient(t *testing.T) {
	t.Run("direct returns the shared client", func(t *testing.T) {
		c, err := buildFetchClient(FetchOptions{Mode: FetchModeDirect})
		if err != nil {
			t.Fatalf("direct: unexpected error %v", err)
		}
		if c != httpClient {
			t.Errorf("direct should reuse the shared httpClient")
		}
	})

	t.Run("clean_dns returns a dedicated client", func(t *testing.T) {
		c, err := buildFetchClient(FetchOptions{Mode: FetchModeCleanDNS})
		if err != nil {
			t.Fatalf("clean_dns: unexpected error %v", err)
		}
		if c == nil || c == httpClient {
			t.Errorf("clean_dns should build a separate client with a custom dialer")
		}
	})

	t.Run("proxy requires a url", func(t *testing.T) {
		if _, err := buildFetchClient(FetchOptions{Mode: FetchModeProxy}); err == nil {
			t.Error("proxy with empty URL should error")
		}
		c, err := buildFetchClient(FetchOptions{Mode: FetchModeProxy, ProxyURL: "socks5://127.0.0.1:7893"})
		if err != nil || c == nil || c == httpClient {
			t.Errorf("proxy with URL should build a separate client, got client=%v err=%v", c, err)
		}
	})

	t.Run("unknown mode errors", func(t *testing.T) {
		if _, err := buildFetchClient(FetchOptions{Mode: "bogus"}); err == nil {
			t.Error("unknown mode should error")
		}
	})
}
