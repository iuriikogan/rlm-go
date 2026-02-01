package rlm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/iuriikogan/rlm-go/internal/client"
	"github.com/iuriikogan/rlm-go/internal/env"
	"github.com/iuriikogan/rlm-go/internal/observability"
	"github.com/iuriikogan/rlm-go/internal/types"
	"github.com/iuriikogan/rlm-go/internal/utils"
)

// RLM represents the Recursive Language Model engine.
// It orchestrates the interaction between the LLM and the code execution environment (REPL).
type RLM struct {
	client        client.Client
	maxIterations int
	systemPrompt  string
}

// NewRLM creates a new RLM instance with the given client and maximum iteration count.
func NewRLM(c client.Client, maxIter int) *RLM {
	return &RLM{
		client:        c,
		maxIterations: maxIter,
		systemPrompt: `You are a Recursive Language Model. You are tasked with answering a query with associated context. You can access, transform, and analyze this context interactively in a REPL environment that can recursively query sub-LLMs, which you are strongly encouraged to use as much as possible. You will be queried iteratively until you provide a final answer.

Your context is available in the 'context' variable.
Context Type: %s
Context Total Length: %d characters

The REPL environment is initialized with:
1. A 'context' variable that contains extremely important information about your query. You should check the content of the 'context' variable to understand what you are working with.
2. A 'llm_query' function that allows you to query an LLM inside your REPL environment. Use it like: llm_query("your question") or llm_query("question", model="model-name").
3. The ability to use 'print()' statements to view the output of your REPL code and continue your reasoning.

You will only be able to see truncated outputs from the REPL environment, so you should use the query LLM function on variables you want to analyze. Use these variables as buffers to build up your final answer.
Make sure to explicitly look through the entire context in REPL before answering your query.

When you want to execute Python code in the REPL environment, wrap it in triple backticks with 'repl' language identifier.

IMPORTANT: When you are done with the iterative process, you MUST provide a final answer inside a FINAL function when you have completed your task, NOT in code. Do not use these tags unless you have completed your task. You have two options:
1. Use FINAL(your final answer here) to provide the answer directly
2. Use FINAL_VAR(variable_name) to return a variable you have created in the REPL environment as your final output

Think step by step carefully, plan, and execute this plan immediately in your response. Output to the REPL environment and recursive LLMs as much as possible.`,
	}
}

// Completion executes the RLM loop to answer the provided prompt.
// It iteratively generates code, executes it in a REPL, and feeds the result back to the LLM
// until a final answer is reached or maxIterations is exceeded.
func (r *RLM) Completion(ctx context.Context, prompt string, contextData interface{}) (*types.RLMChatCompletion, error) {
	start := time.Now()
	defer func() {
		observability.RlmDuration.Observe(time.Since(start).Seconds())
	}()

	slog.Info("Starting RLM completion", "prompt_len", len(prompt))

	// 1. Setup REPL and Handler
	replProxy := &replProxy{}
	handler := NewLMHandler(r.client, replProxy)
	addr, err := handler.Start()
	if err != nil {
		observability.RlmErrors.Inc()
		slog.Error("Failed to start LMHandler", "error", err)
		return nil, err
	}
	defer handler.Stop()

	pythonRepl, err := env.NewPythonREPL(ctx, addr)
	if err != nil {
		observability.RlmErrors.Inc()
		slog.Error("Failed to start PythonREPL", "error", err)
		return nil, err
	}
	defer pythonRepl.Cleanup()
	replProxy.repl = pythonRepl

	// 2. Initialize REPL with context
	// If contextData is provided, use it. Otherwise, use prompt as context if appropriate, 
	// but strictly speaking prompt is the query. 
	// The paper implies "user prompt" is the big thing. 
	// Here we separate query (prompt) and data (contextData).
	
	targetData := contextData
	if targetData == nil {
		// If no context data provided, maybe the prompt IS the context?
		// For now let's assume empty context or just pass nil.
		// Actually, let's pass the prompt as context if contextData is nil, 
		// assuming single-string input mode.
		targetData = prompt 
	}

	contextJSON, err := json.Marshal(targetData)
	if err != nil {
		slog.Error("Failed to marshal context data", "error", err)
		return nil, err
	}
	
	// We need to escape the JSON string to be a valid Python string literal.
	// Python: context = json.loads('...') 
	// Go's json.Marshal returns bytes. String(bytes) gives us the JSON string.
	// We need to escape ' and \ inside that string.
	// Or use triple quotes: context = json.loads("""...""")
	
	// just set it as a string if it's a string, or dict if dict.

	jsonStr := string(contextJSON)
	// Escape triple quotes if present (rare in json, but possible)
	// Actually, just loading it as a raw string is easiest if we don't have quote issues.

	// To adhere strictly to "findings", we need "symbolic handle".
	// The variable `context` IS the symbolic handle.
	
	initCode := fmt.Sprintf("import json; context = json.loads(%q)", jsonStr)
	pythonRepl.ExecuteCode(ctx, initCode)

	// 3. Prepare System Prompt with Metadata
	contextType := "Data"
	contextLen := len(jsonStr)
	if contextData == nil {
		contextType = "String" // or whatever prompt is
	} else {
		contextType = fmt.Sprintf("%T", contextData)
	}

	formattedSystemPrompt := fmt.Sprintf(r.systemPrompt, contextType, contextLen)

	// 4. Main loop
	// User prompt is the query.
	messages := []types.Message{
		{Role: "system", Content: formattedSystemPrompt},
		{Role: "user", Content: fmt.Sprintf("Query: %s", prompt)},
	}

	for i := 0; i < r.maxIterations; i++ {
		if ctx.Err() != nil {
			slog.Warn("Context cancelled", "iteration", i)
			return nil, ctx.Err()
		}

		slog.Debug("RLM Iteration", "iteration", i)

		resp, err := r.client.Completion(ctx, messages)
		if err != nil {
			observability.RlmErrors.Inc()
			slog.Error("Client completion failed", "error", err)
			return nil, err
		}

		messages = append(messages, types.Message{Role: "assistant", Content: resp})

		// Parse code blocks
		codeBlocks := utils.FindCodeBlocks(resp)
		var results []types.CodeBlock
		for _, code := range codeBlocks {
			res := pythonRepl.ExecuteCode(ctx, code)
			results = append(results, types.CodeBlock{Code: code, Result: res})

			// Feed back results (Truncated per RLM paper findings to prevent context pollution)
			stdout := res.Stdout
			if len(stdout) > 2000 {
				stdout = stdout[:2000] + "\n...[Output Truncated]..."
			}
			stderr := res.Stderr
			if len(stderr) > 2000 {
				stderr = stderr[:2000] + "\n...[Error Truncated]..."
			}

			feedback := fmt.Sprintf("REPL Output:\nStdout: %s\nStderr: %s", stdout, stderr)
			messages = append(messages, types.Message{Role: "user", Content: feedback})
		}

		// Check for final answer
		final := utils.FindFinalAnswer(resp)
		if final != "" {
			observability.RlmIterations.Observe(float64(i + 1))
			slog.Info("RLM finished with answer", "iterations", i+1)
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

	observability.RlmIterations.Observe(float64(r.maxIterations))
	slog.Warn("RLM reached max iterations", "max_iterations", r.maxIterations)
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
