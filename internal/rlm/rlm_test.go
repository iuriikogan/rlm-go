package rlm

import (
	"context"
	"strings"
	"testing"

	"github.com/iuriikogan/rlm-go/internal/types"
)

type MockClient struct {
	responses []string
	callCount int
}

func (m *MockClient) Completion(ctx context.Context, messages []types.Message) (string, error) {
	if m.callCount >= len(m.responses) {
		return "FINAL(Mocked Limit Reached)", nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return resp, nil
}

func (m *MockClient) GetUsageSummary() types.UsageSummary {
	return types.UsageSummary{}
}

func (m *MockClient) ModelName() string {
	return "mock-model"
}

func TestRLM_Completion(t *testing.T) {
	tests := []struct {
		name          string
		responses     []string
		prompt        string
		expected      string
		maxIterations int
	}{
		{
			name: "Simple answer",
			responses: []string{
				"FINAL(42)",
			},
			prompt:        "What is the answer?",
			expected:      "42",
			maxIterations: 5,
		},
		{
			name: "One iteration with REPL",
			responses: []string{
				"```repl\nprint('hello')\n```",
				"FINAL(hello)",
			},
			prompt:        "Print hello",
			expected:      "hello",
			maxIterations: 5,
		},
		{
			name: "Max iterations reached",
			responses: []string{
				"Thinking...",
				"Thinking...",
				"Thinking...",
			},
			prompt:        "Loop forever",
			expected:      "Maximum iterations reached without final answer.",
			maxIterations: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockClient{responses: tt.responses}
			rlm := NewRLM(mockClient, tt.maxIterations)

			// We need to skip REPL execution for unit tests or mock it too.
			// Since rlm.go creates a real PythonREPL, this is an integration test unless we abstract REPL creation.
			// For now, we will let it fail on REPL creation if python is not there,
			// BUT wait, NewRLM logic starts the handler and REPL inside Completion.
			// This makes unit testing hard without refactoring.
			// However, if we assume python3 is available (Linux env), it might work.
			// If not, we should refactor RLM to accept a REPL factory.

			// Let's rely on the fact that the mocked responses for "Simple answer"
			// don't trigger REPL code blocks, so REPL creation might happen but not be used
			// if we strip the REPL logic or if the prompt doesn't generate code.
			// BUT RLM.Completion starts REPL unconditionally.

			// For this test to pass without a real Python environment or networking,
			// we should mock the REPL or Handler.
			// Refactoring RLM to accept a REPL factory is the best "Idiomatic Go" way.
			// But for now, let's just try to run it. If it fails, I'll refactor.

			resp, err := rlm.Completion(context.Background(), tt.prompt, nil)
			if err != nil {
				// If we can't start REPL (e.g. no python3), skip test
				if strings.Contains(err.Error(), "executable file not found") {
					t.Skip("Python3 not found")
				}
				t.Fatalf("Completion failed: %v", err)
			}

			if resp.Response != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, resp.Response)
			}
		})
	}
}
