package commands

import (
	"fmt"

	"github.com/eachlabs/klaw/internal/config"
	"github.com/eachlabs/klaw/internal/provider"
)

// initializeProvider selects and initializes the appropriate LLM provider by
// trying providers in order of preference: OpenRouter, then Anthropic.
// Used by commands that don't expose a --provider flag (e.g. node start).
func initializeProvider(cfg *config.Config, model string) (provider.Provider, error) {
	// Try OpenRouter
	if p := cfg.Provider["openrouter"]; p.APIKey != "" {
		prov, err := provider.NewOpenRouter(provider.OpenRouterConfig{
			APIKey:  p.APIKey,
			BaseURL: p.BaseURL,
			Model:   model,
		})
		if err == nil {
			return prov, nil
		}
	}

	// Try Anthropic
	if p := cfg.Provider["anthropic"]; p.APIKey != "" {
		prov, err := provider.NewAnthropic(provider.AnthropicConfig{
			APIKey:  p.APIKey,
			BaseURL: p.BaseURL,
			Model:   model,
		})
		if err == nil {
			return prov, nil
		}
		return nil, fmt.Errorf("failed to initialize Anthropic provider: %w", err)
	}

	return nil, fmt.Errorf(`no LLM provider configured

Please set one of the following:
  - OPENROUTER_API_KEY environment variable
  - ANTHROPIC_API_KEY environment variable
  - Provider configuration in klaw.toml

Example klaw.toml configuration:

    [provider.openrouter]
    api_key = "sk-or-v1-..."
    model = "google/gemini-2.0-flash-001"

    OR

    [provider.anthropic]
    api_key = "sk-ant-..."
    model = "claude-sonnet-4-20250514"
`)
}

// buildProvider resolves provider name, model, and creates the provider instance.
// It is the single source of truth for provider+model selection in commands that
// accept --provider / --model flags (e.g. chat, start).
//
// name:  requested provider name, empty = auto-detect from configured API keys
// model: requested model, empty = resolve from config then provider defaults
//
// Returns the created provider, the resolved model name, and any error.
func buildProvider(cfg *config.Config, name, model string) (provider.Provider, string, error) {
	// Auto-detect provider when no explicit name given
	if name == "" {
		switch {
		case cfg.Provider["openrouter"].APIKey != "":
			name = "openrouter"
		case cfg.Provider["eachlabs"].APIKey != "":
			name = "eachlabs"
		case cfg.Provider["anthropic"].APIKey != "":
			name = "anthropic"
		default:
			name = "anthropic"
		}
	}

	// Resolve model: flag → provider config → provider default → global default
	if model == "" {
		model = cfg.Provider[name].Model
	}
	if model == "" {
		switch name {
		case "openrouter":
			model = "anthropic/claude-sonnet-4"
		case "eachlabs":
			model = "anthropic/claude-sonnet-4-5"
		default:
			model = cfg.Defaults.Model
			if model == "" {
				model = "claude-sonnet-4-20250514"
			}
		}
	}

	// Create provider
	p := cfg.Provider[name]
	var (
		prov provider.Provider
		err  error
	)
	switch name {
	case "openrouter":
		if p.APIKey == "" {
			return nil, "", fmt.Errorf("openrouter API key not set (OPENROUTER_API_KEY or [provider.openrouter] api_key)")
		}
		prov, err = provider.NewOpenRouter(provider.OpenRouterConfig{
			APIKey:  p.APIKey,
			BaseURL: p.BaseURL,
			Model:   model,
		})
	case "eachlabs":
		if p.APIKey == "" {
			return nil, "", fmt.Errorf("eachlabs API key not set (EACHLABS_API_KEY or [provider.eachlabs] api_key)")
		}
		prov, err = provider.NewEachLabs(provider.EachLabsConfig{
			APIKey: p.APIKey,
			Model:  model,
		})
	default: // anthropic
		if p.APIKey == "" {
			return nil, "", fmt.Errorf("anthropic API key not set (ANTHROPIC_API_KEY or [provider.anthropic] api_key)")
		}
		prov, err = provider.NewAnthropic(provider.AnthropicConfig{
			APIKey:  p.APIKey,
			BaseURL: p.BaseURL,
			Model:   model,
		})
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to create %s provider: %w", name, err)
	}
	return prov, model, nil
}
