package handler

import (
	"strings"
	"testing"

	"minioa/internal/service"
)

func TestAIUsageObserverProtocols(t *testing.T) {
	largeText := strings.Repeat("a", 25_000)
	tests := []struct {
		name, path, body string
		stream           bool
		want             *service.AITokenUsage
	}{
		{"chat JSON", "/chat/completions", `{"choices":[{"usage":{"prompt_tokens":99,"completion_tokens":99}}],"usage":{"prompt_tokens":2000,"completion_tokens":500}}`, false, &service.AITokenUsage{Input: 2000, Output: 500}},
		{"embeddings JSON", "/embeddings", `{"data":[{"embedding":[1,2.3,-14.5]}],"usage":{"prompt_tokens":25,"total_tokens":25}}`, false, &service.AITokenUsage{Input: 25}},
		{"responses JSON", "/responses", `{"usage":{"input_tokens":5,"output_tokens":7},"output":[]}`, false, &service.AITokenUsage{Input: 5, Output: 7}},
		{"responses SSE", "/responses", "event: response.completed\r\ndata: {\"type\":\"response.completed\",\r\ndata: \"response\":{\"output\":[{\"text\":\"" + largeText + "\"}],\"usage\":{\"input_tokens\":31,\"output_tokens\":41}}}\r\n\r\n", true, &service.AITokenUsage{Input: 31, Output: 41}},
		{"chat SSE", "/chat/completions", ": heartbeat\n\ndata: {\"usage\":null}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":0}}\n\ndata: [DONE]\n\n", true, &service.AITokenUsage{Input: 4}},
		{"responses nonterminal SSE", "/responses", "data: {\"type\":\"response.created\",\"response\":{\"usage\":{\"input_tokens\":4,\"output_tokens\":6}}}\n\n", true, nil},
		{"incomplete SSE event", "/chat/completions", "data: {\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":0}}\n", true, nil},
		{"missing field", "/chat/completions", `{"usage":{"prompt_tokens":50}}`, false, nil},
		{"negative field", "/responses", `{"usage":{"input_tokens":-1,"output_tokens":50}}`, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := observeAIUsage(tt.path, tt.stream)
			// Split prefixes, line endings and JSON numbers across reads.
			for body := tt.body; body != ""; {
				n := min(13, len(body))
				observer.observe([]byte(body[:n]))
				body = body[n:]
			}
			got := observer.finish()
			if (got == nil) != (tt.want == nil) || got != nil && *got != *tt.want {
				t.Fatalf("usage = %+v, want %+v", got, tt.want)
			}
		})
	}
}
