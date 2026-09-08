package provider

import "github.com/nikhilsaxena04/omni-router/config"

type DeepSeekProvider struct {
	*OpenAIProvider
}

func NewDeepSeekProvider(cfg *config.ProviderConfig) *DeepSeekProvider {
	return &DeepSeekProvider{
		OpenAIProvider: NewOpenAIProvider(cfg),
	}
}

func (p *DeepSeekProvider) Name() string {
	return "deepseek"
}
