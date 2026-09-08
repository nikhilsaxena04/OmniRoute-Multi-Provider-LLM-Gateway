package provider

import (
	"context"
)

type Response struct {
	Text         string
	InputTokens  int
	OutputTokens int
	Cost         float64 // Calculated at the routing/handler layer
	Provider     string
}

type Chunk struct {
	Text string
}

type Provider interface {
	Name() string
	Complete(ctx context.Context, prompt string) (*Response, error)
	Stream(ctx context.Context, prompt string, out chan<- Chunk) error
}
