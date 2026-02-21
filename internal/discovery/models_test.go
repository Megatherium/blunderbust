// Copyright (C) 2026 megatherium
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestIsProviderActive(t *testing.T) {
	registry, err := NewRegistry("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	provider := Provider{
		ID:  "test-provider",
		Env: []string{"TEST_API_KEY"},
	}

	// Should be inactive initially
	if registry.isProviderActive(provider) {
		t.Errorf("expected provider to be inactive when env var is missing")
	}

	// Should be active when env var is set using t.Setenv for isolation
	t.Setenv("TEST_API_KEY", "dummy")
	if !registry.isProviderActive(provider) {
		t.Errorf("expected provider to be active when env var is set")
	}
}

func TestGetActiveModels(t *testing.T) {
	registry, err := NewRegistry("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	providers := map[string]Provider{
		"p1": {
			ID:  "p1",
			Env: []string{"P1_KEY"},
			Models: map[string]Model{
				"m1": {ID: "m1", Name: "Model 1"},
			},
		},
		"p2": {
			ID:  "p2",
			Env: []string{"P2_KEY"},
			Models: map[string]Model{
				"m2": {ID: "m2", Name: "Model 2"},
			},
		},
	}
	registry.SetProviders(providers)

	t.Setenv("P1_KEY", "val")
	// P2_KEY is not set

	active := registry.GetActiveModels()
	if len(active) != 1 {
		t.Fatalf("expected 1 active model, got %d", len(active))
	}

	if active[0] != "p1/m1" {
		t.Errorf("expected active model p1/m1, got %s", active[0])
	}
}

func TestGetModelsForProvider(t *testing.T) {
	registry, err := NewRegistry("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	providers := map[string]Provider{
		"p1": {
			ID: "p1",
			Models: map[string]Model{
				"m1": {ID: "m1", Name: "Model 1"},
				"m2": {ID: "m2", Name: "Model 2"},
			},
		},
	}
	registry.SetProviders(providers)

	models := registry.GetModelsForProvider("p1")
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	missing := registry.GetModelsForProvider("nonexistent")
	if len(missing) != 0 {
		t.Fatalf("expected 0 models for nonexistent provider, got %d", len(missing))
	}
}

func TestRefreshContextCancellation(t *testing.T) {
	// A server that hangs to ensure we hit the context cancellation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	registry, _ := NewRegistry(t.TempDir())

	// Override the URL to our test server
	// Note: Refresh uses a hardcoded URL, so we can't test it directly hitting the server.
	// We'll test standard cancellation. We need to mock the client's transport to redirect.
	registry.client.Transport = &rewriteTransport{server.URL}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := registry.Refresh(ctx)
	if err == nil {
		t.Fatal("expected error due to context timeout, got nil")
	}
	if err.Error() != "context deadline exceeded" && !contains(err.Error(), "context deadline exceeded") && !contains(err.Error(), "context canceled") {
		t.Errorf("expected context timeout error, got: %v", err)
	}
}

func TestRefreshErrors(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		errMatches string
	}{
		{
			name: "server error 500",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			errMatches: "unexpected status from models.dev",
		},
		{
			name: "invalid json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{invalid json`))
			},
			errMatches: "decoding api.json",
		},
		{
			name: "empty json object",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{}`))
			},
			errMatches: "validation error",
		},
		{
			name: "missing provider ID",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"test": {"name": "test"}}`))
			},
			errMatches: "missing ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			registry, _ := NewRegistry(t.TempDir())
			registry.client.Transport = &rewriteTransport{server.URL}

			err := registry.Refresh(context.Background())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tt.errMatches) {
				t.Errorf("expected error containing %q, got: %v", tt.errMatches, err)
			}
		})
	}
}

func TestLoadCacheHandling(t *testing.T) {
	cacheDir := t.TempDir()
	registry, _ := NewRegistry(cacheDir)

	// Test 1: Load missing cache -> triggers Refresh.
	// We mock refresh to fail to see if the missing cache properly calls it.
	registry.client.Transport = &errorTransport{}
	err := registry.Load(context.Background())
	if err == nil || !contains(err.Error(), "simulated network error") {
		t.Errorf("expected network error from refresh on missing cache, got: %v", err)
	}

	// Test 2: Cache exists but is corrupt -> triggers Refresh.
	if err := os.WriteFile(registry.GetCachePath(), []byte("corrupt json"), 0o600); err != nil {
		t.Fatalf("failed to write corrupt cache: %v", err)
	}
	err = registry.Load(context.Background())
	if err == nil || !contains(err.Error(), "cache corrupted") {
		t.Errorf("expected cache corrupted error, got: %v", err)
	}

	// Test 3: Cache exists and is valid.
	validData := map[string]Provider{
		"test": {ID: "test"},
	}
	b, _ := json.Marshal(validData)
	if err := os.WriteFile(registry.GetCachePath(), b, 0o600); err != nil {
		t.Fatalf("failed to write valid cache: %v", err)
	}

	err = registry.Load(context.Background())
	if err != nil {
		t.Errorf("expected success loading valid cache, got: %v", err)
	}

	registry.mu.RLock()
	if len(registry.providers) != 1 {
		t.Errorf("expected 1 provider loaded from cache, got %d", len(registry.providers))
	}
	registry.mu.RUnlock()
}

// Helpers

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s[len(s)-len(substr):] == substr ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

// rewriteTransport overrides the request URL to hit the test server instead of models.dev
type rewriteTransport struct {
	targetURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := req.Clone(req.Context())
	// Extract scheme and host from targetURL
	// This is a simplified rewrite for testing
	newReq.URL.Scheme = "http"
	newReq.URL.Host = t.targetURL[7:] // strip "http://"
	return http.DefaultTransport.RoundTrip(newReq)
}

// errorTransport always returns a network error
type errorTransport struct{}

func (t *errorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("simulated network error")
}
