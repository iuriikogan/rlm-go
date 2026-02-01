package client

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/iuriikogan/rlm-go/internal/observability"
	"github.com/iuriikogan/rlm-go/internal/types"
	"google.golang.org/genai"
)

type Client interface {
	Completion(ctx context.Context, messages []types.Message) (string, error)
	GetUsageSummary() types.UsageSummary
	ModelName() string
}

type GeminiClient struct {
	client    *genai.Client
	modelName string
	usage     types.UsageSummary
}

func NewGeminiClient(apiKey string, modelName string) (*GeminiClient, error) {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required")
	}

	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}

	return &GeminiClient{
		client:    client,
		modelName: modelName,
	}, nil
}

func (c *GeminiClient) Completion(ctx context.Context, messages []types.Message) (string, error) {
	// Convert messages to Gemini format
	var contents []*genai.Content
	var systemInstruction *genai.Content

	for _, msg := range messages {
		if msg.Role == "system" {
			systemInstruction = &genai.Content{
				Parts: []*genai.Part{
					{Text: msg.Content},
				},
			}
		} else {
			role := msg.Role
			if role == "assistant" {
				role = "model"
			}
			contents = append(contents, &genai.Content{
				Role: role,
				Parts: []*genai.Part{
					{Text: msg.Content},
				},
			})
		}
	}

	config := &genai.GenerateContentConfig{
		// MaxOutputTokens left unset (0) to use model default/unlimited
	}
	if systemInstruction != nil {
		config.SystemInstruction = systemInstruction
	}

	resp, err := c.client.Models.GenerateContent(ctx, c.modelName, contents, config)
	if err != nil {
		slog.Error("Gemini API call failed", "error", err, "model", c.modelName)
		return "", err
	}

	// Track usage
	if resp.UsageMetadata != nil {
		inputTokens := int(resp.UsageMetadata.PromptTokenCount)
		outputTokens := int(resp.UsageMetadata.CandidatesTokenCount)

		c.usage.TotalCalls++
		c.usage.TotalInputTokens += inputTokens
		c.usage.TotalOutputTokens += outputTokens

		observability.TokenUsage.WithLabelValues(c.modelName, "input").Add(float64(inputTokens))
		observability.TokenUsage.WithLabelValues(c.modelName, "output").Add(float64(outputTokens))
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		slog.Warn("No response content from model", "model", c.modelName)
		return "", fmt.Errorf("no response from model")
	}

	return resp.Candidates[0].Content.Parts[0].Text, nil
}

func (c *GeminiClient) GetUsageSummary() types.UsageSummary {
	return c.usage
}

func (c *GeminiClient) ModelName() string {
	return c.modelName
}
