package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestApiKeySentAsBearer(t *testing.T) {
	var seenAuth []string
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/graphql" {
			t.Errorf("unexpected request to %s; no token exchange should happen", r.URL.Path)
		}
		seenAuth = append(seenAuth, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"ok": true}}`))
	}))
	defer apiSrv.Close()

	client := NewClient(WithEndpoint(apiSrv.URL+"/"), WithApiKey("sk-nb-test"))
	for i := 0; i < 2; i++ {
		var resp map[string]any
		if err := client.Run(context.Background(), NewRequest(`query { ok }`), &resp); err != nil {
			t.Fatalf("run %d failed: %v", i, err)
		}
	}

	if len(seenAuth) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(seenAuth))
	}
	for _, auth := range seenAuth {
		if auth != "Bearer sk-nb-test" {
			t.Fatalf("expected raw API key as Bearer, got %q", auth)
		}
	}
}

func TestUnauthorizedReturnsHint(t *testing.T) {
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Unauthorized"}`))
	}))
	defer apiSrv.Close()

	t.Run("sk-nb key", func(t *testing.T) {
		client := NewClient(WithEndpoint(apiSrv.URL), WithApiKey("sk-nb-revoked"))
		err := client.Run(context.Background(), NewRequest(`query { ok }`), nil)
		if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "Create a new token") {
			t.Fatalf("expected 401 hint, got %v", err)
		}
		if strings.Contains(err.Error(), "must start with") {
			t.Fatalf("did not expect prefix hint for sk-nb key, got %v", err)
		}
	})

	t.Run("legacy key", func(t *testing.T) {
		client := NewClient(WithEndpoint(apiSrv.URL), WithApiKey("old-secret"))
		err := client.Run(context.Background(), NewRequest(`query { ok }`), nil)
		if err == nil || !strings.Contains(err.Error(), `must start with "sk-nb-"`) {
			t.Fatalf("expected prefix hint, got %v", err)
		}
	})
}

func TestMissingApiKey(t *testing.T) {
	viper.Set("api-key", "")
	defer viper.Set("api-key", nil)
	called := false
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer apiSrv.Close()

	err := NewClient(WithEndpoint(apiSrv.URL)).Run(context.Background(), NewRequest(`query { ok }`), nil)
	if err == nil || !strings.Contains(err.Error(), "no API key configured") {
		t.Fatalf("expected missing key error, got %v", err)
	}
	if called {
		t.Fatal("request should not be sent without an API key")
	}
}

func TestNewClient(t *testing.T) {
	t.Run("with options", func(t *testing.T) {
		client := NewClient(
			WithEndpoint("http://test.com"),
			WithApiKey("test-key"),
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
		// no need to reset viper as it's global and tests run in sequence or we just let it be
		// better to use cleanup if we care about other tests, but here we are ok.

		client := NewClient()
		if client.endpoint != "http://viper.com/api/graphql" {
			t.Errorf("expected endpoint to be http://viper.com/api/graphql, got %s", client.endpoint)
		}
	})
}
