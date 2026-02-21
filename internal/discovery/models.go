package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	// PrefixProvider is the prefix used to specify all models from a specific provider.
	PrefixProvider = "provider:"
	// KeywordDiscoverActive is the keyword used to dynamically include all models from all active providers.
	KeywordDiscoverActive = "discover:active"
)

// Provider represents an LLM provider from models.dev/api.json
type Provider struct {
	// ID is the unique identifier for the provider (e.g., "openai", "anthropic").
	ID string `json:"id"`
	// Name is the display name of the provider.
	Name string `json:"name"`
	// Env contains the list of environment variables required to activate this provider.
	Env []string `json:"env"`
	// API is the base URL for the provider's API.
	API string `json:"api"`
	// Models maps model IDs to their respective Model configurations.
	Models map[string]Model `json:"models"`
}

// Model represents a specific LLM model.
type Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Registry handles model discovery and caching.
type Registry struct {
	cachePath string
	mu        sync.RWMutex
	providers map[string]Provider
	client    *http.Client
}

// NewRegistry creates a new Registry with a default cache path.
// If cacheDir is empty, it defaults to ~/.cache/blunderbuss.
func NewRegistry(cacheDir string) (*Registry, error) {
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not determine user home directory: %w", err)
		}
		cacheDir = filepath.Join(home, ".cache", "blunderbuss")
	}
	return &Registry{
		cachePath: filepath.Join(cacheDir, "models-api.json"),
		providers: make(map[string]Provider),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// GetCachePath returns the path to the models-api.json cache file.
func (r *Registry) GetCachePath() string {
	return r.cachePath
}

// Refresh fetches the latest api.json from models.dev and updates the cache.
func (r *Registry) Refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://models.dev/api.json", http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	var resp *http.Response
	var fetchErr error

	// Simple retry loop (3 attempts)
	for i := 0; i < 3; i++ {
		resp, fetchErr = r.client.Do(req)
		if fetchErr == nil {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
	if fetchErr != nil {
		return fmt.Errorf("fetching models.dev/api.json: %w", fetchErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status from models.dev: %s", resp.Status)
	}

	var parsed map[string]Provider
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("decoding api.json: %w", err)
	}

	if len(parsed) == 0 {
		return fmt.Errorf("validation error: decoded JSON is empty")
	}

	// Validate basic structure
	for k, v := range parsed {
		if v.ID == "" {
			return fmt.Errorf("validation error: provider %q missing ID", k)
		}
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(filepath.Dir(r.cachePath), 0o750); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	data, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling api.json for cache: %w", err)
	}

	if err := os.WriteFile(r.cachePath, data, 0o600); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	r.mu.Lock()
	r.providers = parsed
	r.mu.Unlock()

	return nil
}

// Load attempts to load providers from the local cache.
// If the cache is missing or corrupt, it triggers a Refresh.
func (r *Registry) Load(ctx context.Context) error {
	data, err := os.ReadFile(r.cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return r.Refresh(ctx) // Fetch if missing
		}
		return fmt.Errorf("reading cache file: %w", err)
	}

	var parsed map[string]Provider
	if err := json.Unmarshal(data, &parsed); err != nil {
		// Cache corruption or invalid JSON, attempt to refresh
		refreshErr := r.Refresh(ctx)
		if refreshErr != nil {
			return fmt.Errorf("cache corrupted (%w) and refresh failed: %v", err, refreshErr)
		}
		return nil
	}

	if len(parsed) == 0 {
		refreshErr := r.Refresh(ctx)
		if refreshErr != nil {
			return fmt.Errorf("cache is empty and refresh failed: %v", refreshErr)
		}
		return nil
	}

	r.mu.Lock()
	r.providers = parsed
	r.mu.Unlock()

	return nil
}

// GetActiveModels returns a list of model IDs from providers that have their required env vars set.
func (r *Registry) GetActiveModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var activeModels []string
	for _, p := range r.providers {
		if r.isProviderActive(p) {
			for _, m := range p.Models {
				// Format as provider/model-id
				activeModels = append(activeModels, fmt.Sprintf("%s/%s", p.ID, m.ID))
			}
		}
	}
	sort.Strings(activeModels)
	return activeModels
}

func (r *Registry) isProviderActive(p Provider) bool {
	if len(p.Env) == 0 {
		return false
	}
	for _, envVar := range p.Env {
		if os.Getenv(envVar) == "" {
			return false
		}
	}
	return true
}

// GetModelsForProvider returns a list of model IDs for a specific provider.
func (r *Registry) GetModelsForProvider(providerID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[providerID]
	if !ok {
		return nil
	}
	var models []string
	for _, m := range p.Models {
		models = append(models, fmt.Sprintf("%s/%s", p.ID, m.ID))
	}
	sort.Strings(models)
	return models
}

// SetProviders is a helper method used for tests to inject mock providers
func (r *Registry) SetProviders(providers map[string]Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = providers
}
