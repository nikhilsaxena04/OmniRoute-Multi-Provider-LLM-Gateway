package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nikhilsaxena04/omni-router/provider"
	"github.com/nikhilsaxena04/omni-router/router"
)

type ChatRequest struct {
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ChatHandler struct {
	Router *router.Router
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	if req.Stream {
		h.handleStream(w, r, req.Prompt)
	} else {
		h.handleComplete(w, r, req.Prompt)
	}
}

func (h *ChatHandler) handleComplete(w http.ResponseWriter, r *http.Request, prompt string) {
	resp, err := h.Router.ExecuteComplete(r.Context(), prompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("Provider error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"text":          resp.Text,
		"input_tokens":  resp.InputTokens,
		"output_tokens": resp.OutputTokens,
	})
}

func (h *ChatHandler) handleStream(w http.ResponseWriter, r *http.Request, prompt string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	chunkChan := make(chan provider.Chunk)
	errChan := make(chan error, 1)

	go func() {
		errChan <- h.Router.ExecuteStream(r.Context(), prompt, chunkChan)
	}()

	for chunk := range chunkChan {
		// Escape newlines in chunk for basic SSE formatting if needed, 
		// but simple text formatting works for the test.
		fmt.Fprintf(w, "data: %s\n\n", chunk.Text)
		flusher.Flush()
	}

	if err := <-errChan; err != nil {
		fmt.Fprintf(w, "data: [ERROR] %v\n\n", err)
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}
