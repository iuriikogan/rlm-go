package rlm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/iuriikogan/rlm-go/pkg/client"
	"github.com/iuriikogan/rlm-go/pkg/env"
	"github.com/iuriikogan/rlm-go/pkg/types"
	"github.com/iuriikogan/rlm-go/pkg/utils"
)

type RLM struct {
	client        client.Client
	maxIterations int
	systemPrompt  string
}

func NewRLM(c client.Client, maxIter int) *RLM {
	return &RLM{
		client:        c,
		maxIterations: maxIter,
		systemPrompt:  "You are a Recursive Language Model. Use ```repl\n...``` blocks to execute code and FINAL(answer) to provide the final result.",
	}
}

func (r *RLM) Completion(ctx context.Context, prompt string, contextData interface{}) (*types.RLMChatCompletion, error) {
	start := time.Now()

	// 1. Setup REPL and Handler
	// We need a dummy REPL first to create the handler, then we can start the REPL with the handler address
	// Actually, we can just start the handler with a pointer to the REPL that we'll set later.
	
	// For simplicity, let's just create a shared state
	replProxy := &replProxy{}
	handler := NewLMHandler(r.client, replProxy)
	addr, err := handler.Start()
	if err != nil {
		return nil, err
	}
	defer handler.Stop()

	pythonRepl, err := env.NewPythonREPL(ctx, addr)
	if err != nil {
		return nil, err
	}
	defer pythonRepl.Cleanup()
	replProxy.repl = pythonRepl

	// 2. Initialize REPL with context
	if contextData != nil {
		contextJSON, _ := json.Marshal(contextData)
		pythonRepl.ExecuteCode(ctx, fmt.Sprintf("context = json.loads('%s')", string(contextJSON)))
	}

	// 3. Main loop
	messages := []types.Message{
		{Role: "system", Content: r.systemPrompt},
		{Role: "user", Content: prompt},
	}

	for i := 0; i < r.maxIterations; i++ {
		resp, err := r.client.Completion(ctx, messages)
		if err != nil {
			return nil, err
		}

		messages = append(messages, types.Message{Role: "assistant", Content: resp})

		// Parse code blocks
		codeBlocks := utils.FindCodeBlocks(resp)
		var results []types.CodeBlock
		for _, code := range codeBlocks {
			res := pythonRepl.ExecuteCode(ctx, code)
			results = append(results, types.CodeBlock{Code: code, Result: res})
			
			// Feed back results
			feedback := fmt.Sprintf("REPL Output:\nStdout: %s\nStderr: %s", res.Stdout, res.Stderr)
			messages = append(messages, types.Message{Role: "user", Content: feedback})
		}

		// Check for final answer
		final := utils.FindFinalAnswer(resp)
		if final != "" {
			return &types.RLMChatCompletion{
				RootModel:     r.client.ModelName(),
				Prompt:        prompt,
				Response:      final,
				UsageSummary:  r.client.GetUsageSummary(),
				ExecutionTime: time.Since(start).Seconds(),
			}, nil
		}
		
		if len(codeBlocks) == 0 {
			// If no code blocks and no final answer, ask the model to provide one or continue
			messages = append(messages, types.Message{Role: "user", Content: "Please continue or provide a FINAL(answer)."})
		}
	}

	return &types.RLMChatCompletion{
		RootModel:     r.client.ModelName(),
		Prompt:        prompt,
		Response:      "Maximum iterations reached without final answer.",
		UsageSummary:  r.client.GetUsageSummary(),
		ExecutionTime: time.Since(start).Seconds(),
	}, nil
}

type replProxy struct {
	repl *env.PythonREPL
}

func (p *replProxy) AddPendingCall(call types.RLMChatCompletion) {
	if p.repl != nil {
		p.repl.AddPendingCall(call)
	}
}
