package config

import "testing"

func TestApplyEnvironmentOverridesUsesOrcaRouterKey(t *testing.T) {
	t.Setenv("ORCAROUTER_API_KEY", "sk-orca-test")

	cfg := NewDefaultConfig()
	ApplyEnvironmentOverrides(&cfg)

	if cfg.APIKey != "sk-orca-test" {
		t.Fatalf("APIKey = %q, want environment value", cfg.APIKey)
	}
	if cfg.Provider != OrcaRouterProvider {
		t.Fatalf("Provider = %q, want %q", cfg.Provider, OrcaRouterProvider)
	}
	if cfg.BaseURL != OrcaRouterBaseURL {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, OrcaRouterBaseURL)
	}
	if cfg.Model != OrcaRouterDefaultModel {
		t.Fatalf("Model = %q, want %q", cfg.Model, OrcaRouterDefaultModel)
	}
}

func TestApplyEnvironmentOverridesPreservesSavedAPIKey(t *testing.T) {
	t.Setenv("ORCAROUTER_API_KEY", "sk-orca-test")

	cfg := Config{
		APIKey:   "saved-key",
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o",
	}
	ApplyEnvironmentOverrides(&cfg)

	if cfg.APIKey != "saved-key" || cfg.Provider != "openai" || cfg.BaseURL != "https://api.openai.com/v1" || cfg.Model != "gpt-4o" {
		t.Fatalf("saved provider configuration was unexpectedly changed: %#v", cfg)
	}
}
