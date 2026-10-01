package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestTokenFetchAndAuthHeader(t *testing.T) {
	// token server returns a token with short expiry
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":  "test-token-1",
			"expiry": 2, // seconds
		})
	}))
	defer tokenSrv.Close()

	// graphQL server checks Authorization header
	var seenToken string
	callCount := 0
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		seenToken = r.Header.Get("Authorization")
		// respond OK
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"ok": true}}`))
	}))
	defer apiSrv.Close()

	// configure viper
	viper.Set("endpoint", tokenSrv.URL) // NewClient will append /api/graphql and /api/token, but tests will override below
	viper.Set("api-key", "dummy-key")

	// Build a client but override transport endpoints to point to test servers
	transport := &authTransport{
		apiKey:        "dummy-key",
		legacy:        true,
		tokenEndpoint: tokenSrv.URL,
		wrapped:       http.DefaultTransport,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
	}
	httpClient := &http.Client{Transport: transport}
	client := &Client{
		endpoint:   apiSrv.URL,
		httpClient: httpClient,
	}

	// make a request - this should trigger token fetch
	req := NewRequest(`query { ok }`)
	var resp map[string]any
	if err := client.Run(context.Background(), req, &resp); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if seenToken != "Bearer test-token-1" {
		t.Fatalf("expected Authorization header to contain fetched token, got %q", seenToken)
	}

	// wait until token expires
	time.Sleep(3 * time.Second)

	// next request should fetch a fresh token from token server (which always returns test-token-1)
	req2 := NewRequest(`query { ok }`)
	if err := client.Run(context.Background(), req2, &resp); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	if callCount < 2 {
		t.Fatalf("expected at least two GraphQL calls, got %d", callCount)
	}
}

func TestRetryOn401(t *testing.T) {
	// token server: first returns token1, then token2
	tokens := []string{"token1", "token2"}
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tkn := tokens[0]
		// rotate
		if len(tokens) > 1 {
			tokens = tokens[1:]
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"token": tkn, "expiry": 60})
	}))
	defer tokenSrv.Close()

	// GraphQL server: first request with token1 responds 401, second with token2 responds 200
	call := 0
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		auth := r.Header.Get("Authorization")
		if call == 1 {
			// first call: token1 -> return 401
			if auth != "Bearer token1" {
				t.Fatalf("expected first call to use token1, got %q", auth)
			}
			w.WriteHeader(401)
			return
		}
		// subsequent calls should carry token2
		if auth != "Bearer token2" {
			t.Fatalf("expected retry to use token2, got %q", auth)
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"ok": true}}`))
	}))
	defer apiSrv.Close()

	transport := &authTransport{
		apiKey:        "dummy-key",
		legacy:        true,
		tokenEndpoint: tokenSrv.URL,
		wrapped:       http.DefaultTransport,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
	}
	httpClient := &http.Client{Transport: transport}
	client := &Client{
		endpoint:   apiSrv.URL,
		httpClient: httpClient,
	}

	req := NewRequest(`query { ok }`)
	var resp map[string]any
	if err := client.Run(context.Background(), req, &resp); err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func TestSendsApiKeyAsBearer(t *testing.T) {
	var paths []string
	var seenAuth string
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		seenAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"ok": true}}`))
	}))
	defer apiSrv.Close()

	client := NewClient(WithEndpoint(apiSrv.URL), WithApiKey("sk-nb-test-key"))

	var resp map[string]any
	if err := client.Run(context.Background(), NewRequest(`query { ok }`), &resp); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// A current server accepts the key itself: no exchange call precedes the request.
	if len(paths) != 1 || paths[0] != "/api/graphql" {
		t.Fatalf("expected a single call to /api/graphql, got %v", paths)
	}
	if seenAuth != "Bearer sk-nb-test-key" {
		t.Fatalf("expected the API key as the bearer token, got %q", seenAuth)
	}
}

func TestFallsBackToExchangeOnOlderServer(t *testing.T) {
	// An older server refuses the key as a bearer but exchanges it for a session token.
	directAttempts, exchanges := 0, 0
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/token":
			exchanges++
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "session-token", "expiry": 3600})
		case "/api/graphql":
			if r.Header.Get("Authorization") != "Bearer session-token" {
				directAttempts++
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			// The retried request must still carry the query.
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "query { ok }") {
				t.Errorf("expected the retried request to carry the query, got %q", body)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data": {"ok": true}}`))
		}
	}))
	defer apiSrv.Close()

	client := NewClient(WithEndpoint(apiSrv.URL), WithApiKey("sk-nb-test-key"), WithUsername("a@b.com"))

	for i := 0; i < 2; i++ {
		var resp map[string]any
		if err := client.Run(context.Background(), NewRequest(`query { ok }`), &resp); err != nil {
			t.Fatalf("run %d failed: %v", i, err)
		}
		if resp["ok"] != true {
			t.Fatalf("run %d: expected data, got %v", i, resp)
		}
	}

	// Once the server is known to need the exchange, later requests go straight to it.
	if directAttempts != 1 || exchanges != 1 {
		t.Fatalf("expected 1 direct attempt and 1 exchange, got %d and %d", directAttempts, exchanges)
	}
}

func TestFallbackResendsBodyUnderConcurrency(t *testing.T) {
	// Several first requests hit an older server at once. Each one is refused,
	// exchanged and re-sent, and every re-send must carry its full body.
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "session-token", "expiry": 3600})
		case "/api/graphql":
			if r.Header.Get("Authorization") != "Bearer session-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data": {"ok": true}}`))
		}
	}))
	defer apiSrv.Close()

	client := NewClient(WithEndpoint(apiSrv.URL), WithApiKey("sk-nb-test-key"), WithUsername("a@b.com"))
	query := `query { ok }` + strings.Repeat(" ", 20000)

	const workers = 8
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() {
			var resp map[string]any
			errs <- client.Run(context.Background(), NewRequest(query), &resp)
		}()
	}
	for i := 0; i < workers; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent fallback request failed: %v", err)
		}
	}
}

func TestUnauthorizedIsAnError(t *testing.T) {
	// The app's 401 body has neither `data` nor `errors`, so decoding it alone
	// would report success with an empty result. A current server no longer has
	// the token endpoint, so the fallback fails too.
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/auth/token" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"not_authenticated","description":"The user does not have an active session"}`))
	}))
	defer apiSrv.Close()

	cases := []struct {
		name   string
		apiKey string
		hint   string
	}{
		{"no key", "", "no API key configured"},
		{"key from before the sk-nb- format", "0123456789abcdef", "predates"},
		{"AI Gateway key", "sk-nb-gw-abc", "AI Gateway key"},
		{"deleted or expired key", "sk-nb-abc", "rejected"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(WithEndpoint(apiSrv.URL), WithApiKey(tc.apiKey))
			// NewClient falls back to viper for an empty key.
			client.apiKey = tc.apiKey

			var resp map[string]any
			err := client.Run(context.Background(), NewRequest(`query { ok }`), &resp)
			if err == nil {
				t.Fatal("expected a 401 to be an error")
			}
			if !strings.Contains(err.Error(), tc.hint) {
				t.Fatalf("expected error to mention %q, got %q", tc.hint, err.Error())
			}
		})
	}
}

func TestVerboseLogOmitsApiKey(t *testing.T) {
	t.Chdir(t.TempDir())
	viper.Set("verbose", true)
	defer viper.Set("verbose", false)

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"ok": true}}`))
	}))
	defer apiSrv.Close()

	client := NewClient(WithEndpoint(apiSrv.URL), WithApiKey("sk-nb-secret-key"))
	var resp map[string]any
	if err := client.Run(context.Background(), NewRequest(`query { ok }`), &resp); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	logged, err := os.ReadFile("nbctl_graphql.log")
	if err != nil {
		t.Fatalf("expected a verbose log file: %v", err)
	}
	if !strings.Contains(string(logged), "POST /api/graphql") {
		t.Fatalf("expected the request to be logged, got %q", logged)
	}
	if strings.Contains(string(logged), "sk-nb-secret-key") {
		t.Fatal("the verbose log must not contain the API key")
	}
}

func TestNewClient(t *testing.T) {
	t.Run("with options", func(t *testing.T) {
		client := NewClient(
			WithEndpoint("http://test.com"),
			WithApiKey("test-key"),
			WithUsername("test-user"),
		)
		if client == nil {
			t.Fatal("expected client to be non-nil")
		}
		if client.endpoint != "http://test.com/api/graphql" {
			t.Errorf("expected endpoint to be http://test.com/api/graphql, got %s", client.endpoint)
		}
	})

	t.Run("with viper", func(t *testing.T) {
		viper.Set("endpoint", "http://viper.com")
		viper.Set("api-key", "viper-key")
		viper.Set("username", "viper-user")
		// no need to reset viper as it's global and tests run in sequence or we just let it be
		// better to use cleanup if we care about other tests, but here we are ok.

		client := NewClient()
		if client.endpoint != "http://viper.com/api/graphql" {
			t.Errorf("expected endpoint to be http://viper.com/api/graphql, got %s", client.endpoint)
		}
	})
}
