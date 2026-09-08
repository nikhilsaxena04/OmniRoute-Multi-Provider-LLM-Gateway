package provider

import "github.com/nikhilsaxena04/omni-router/config"

type DeepSeekProvider struct {
	*OpenAIProvider
	name string
}

func NewDeepSeekProvider(cfg *config.ProviderConfig, name string) *DeepSeekProvider {
	return &DeepSeekProvider{
		OpenAIProvider: NewOpenAIProvider(cfg, name),
		name:           name,
	}
}

func (p *DeepSeekProvider) Name() string {
	return p.name
}
