package client

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestVerboseLogRedactsCredentials(t *testing.T) {
	const apiKey = "sk-nb-secret-value"
	var seenAuth, seenBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		seenBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Set-Cookie", "session=cookie-secret")
		_, _ = w.Write([]byte(`{"data": {"ok": true}}`))
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	transport := &loggingTransport{
		wrapped: &authTransport{apiKey: apiKey, wrapped: http.DefaultTransport},
		logger:  log.New(&logBuf, "", 0),
	}
	c := &Client{endpoint: srv.URL + "/api/graphql", apiKey: apiKey, httpClient: &http.Client{Transport: transport}}

	req := NewRequest(`query { ok }`)
	req.Header("Authorization", "Bearer "+apiKey) // a caller-set header must be redacted too
	req.Header("Cookie", "session=cookie-secret")
	var resp map[string]any
	if err := c.Run(context.Background(), req, &resp); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if seenAuth != "Bearer "+apiKey {
		t.Fatalf("server should still get the real Authorization header, got %q", seenAuth)
	}
	if !strings.Contains(seenBody, "query { ok }") {
		t.Fatalf("server should get the full request body, got %q", seenBody)
	}
	logged := logBuf.String()
	for _, secret := range []string{apiKey, "cookie-secret"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("verbose log contains %q:\n%s", secret, logged)
		}
	}
	if !strings.Contains(logged, "[REDACTED]") || !strings.Contains(logged, "query { ok }") {
		t.Fatalf("verbose log should keep the request with redacted headers:\n%s", logged)
	}
}

func TestHTTPTimeout(t *testing.T) {
	defer viper.Set("http-timeout", "")
	cases := map[string]time.Duration{
		"":     DefaultHTTPTimeout,
		"50s":  50 * time.Second,
		"2m":   2 * time.Minute,
		"45":   45 * time.Second,
		"0":    0,
		"nope": DefaultHTTPTimeout,
		"-5s":  DefaultHTTPTimeout,
	}
	for raw, want := range cases {
		viper.Set("http-timeout", raw)
		if got := httpTimeout(); got != want {
			t.Errorf("http-timeout %q: got %s, want %s", raw, got, want)
		}
	}

	viper.Set("http-timeout", "55s")
	if got := NewHTTPClient(WithApiKey("k")).Timeout; got != 55*time.Second {
		t.Errorf("client timeout: got %s, want 55s", got)
	}
}
