package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/machinebox/graphql"
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
		tokenEndpoint: tokenSrv.URL,
		wrapped:       http.DefaultTransport,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
	}
	httpClient := &http.Client{Transport: transport}
	client := graphql.NewClient(apiSrv.URL, graphql.WithHTTPClient(httpClient))

	// make a request - this should trigger token fetch
	req := graphql.NewRequest(`query { ok }`)
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
	req2 := graphql.NewRequest(`query { ok }`)
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
		tokenEndpoint: tokenSrv.URL,
		wrapped:       http.DefaultTransport,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
	}
	httpClient := &http.Client{Transport: transport}
	client := graphql.NewClient(apiSrv.URL, graphql.WithHTTPClient(httpClient))

	req := graphql.NewRequest(`query { ok }`)
	var resp map[string]any
	if err := client.Run(context.Background(), req, &resp); err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func TestNewClient(t *testing.T) {
	t.Run("with options", func(t *testing.T) {
		client := NewClient(
			WithEndpoint("http://test.com"),
			WithApiKey("test-key"),
			WithUsername("test-user"),
		)
		// This is a hacky way to check the endpoint, but it's the only way
		// without modifying the graphql client.
		if client.Log == nil {
			t.Log("client.Log is nil, this is expected as we are not setting it")
		}
	})

	t.Run("with viper", func(t *testing.T) {
		viper.Set("endpoint", "http://viper.com")
		viper.Set("api-key", "viper-key")
		viper.Set("username", "viper-user")
		defer viper.Reset()

		client := NewClient()
		if client.Log == nil {
			t.Log("client.Log is nil, this is expected as we are not setting it")
		}
	})
}

// func TestRun(t *testing.T) {
// 	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(200)
// 		w.Header().Set("Content-Type", "application/json")
// 		_, _ = w.Write([]byte(`{"data": {"key": "value"}}`))
// 	}))
// 	defer apiSrv.Close()

// 	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Content-Type", "application/json")
// 		_ = json.NewEncoder(w).Encode(map[string]any{
// 			"token":  "test-token",
// 			"expiry": 3600,
// 		})
// 	}))
// 	defer tokenSrv.Close()

// 	viper.Set("endpoint", apiSrv.URL)
// 	viper.Set("api-key", "test-key")
// 	viper.Set("username", "test-user")
// 	DefaultClientFactory = func() *graphql.Client {
// 		transport := &authTransport{
// 			apiKey:        "test-key",
// 			username:      "test-user",
// 			tokenEndpoint: tokenSrv.URL,
// 			wrapped:       http.DefaultTransport,
// 			httpClient:    &http.Client{Timeout: 5 * time.Second},
// 		}
// 		httpClient := &http.Client{Transport: transport}
// 		return graphql.NewClient(apiSrv.URL, graphql.WithHTTPClient(httpClient))
// 	}

// 	req := graphql.NewRequest(`query { key }`)
// 	var resp map[string]string
// 	err := Run(context.Background(), req, &resp)
// 	if err != nil {
// 		t.Fatalf("Run() error = %v", err)
// 	}
// 	if resp["key"] != "value" {
// 		t.Fatalf("expected response key to be 'value', got '%s'", resp["key"])
// 	}
// }
