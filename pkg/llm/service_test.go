package llm

import (
	"testing"

	"Q-Solver/pkg/config"
)

func TestDetectProviderType(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     ProviderType
	}{
		{name: "OrcaRouter", provider: "orcarouter", want: ProviderOrcaRouter},
		{name: "case insensitive OrcaRouter", provider: "OrcaRouter", want: ProviderOrcaRouter},
		{name: "Google", provider: "google", want: ProviderGemini},
		{name: "Anthropic", provider: "anthropic", want: ProviderClaude},
		{name: "custom", provider: "custom", want: ProviderCustom},
		{name: "OpenAI fallback", provider: "openai", want: ProviderOpenAI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectProviderType(tt.provider); got != tt.want {
				t.Fatalf("DetectProviderType(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestCreateProviderForOrcaRouterUsesOpenAIAdapter(t *testing.T) {
	provider := CreateProvider(ProviderOrcaRouter, &config.Config{
		APIKey:  "test-key",
		Model:   "openai/gpt-4o-mini",
		BaseURL: "https://api.orcarouter.ai/v1",
	})

	if _, ok := provider.(*OpenAIAdapter); !ok {
		t.Fatalf("CreateProvider(ProviderOrcaRouter) returned %T, want *OpenAIAdapter", provider)
	}
}
