package client

import (
	"context"
	"fmt"
	"os"

	"github.com/iuriikogan/rlm-go/pkg/types"
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

	config := &genai.GenerateContentConfig{}
	if systemInstruction != nil {
		config.SystemInstruction = systemInstruction
	}

	resp, err := c.client.Models.GenerateContent(ctx, c.modelName, contents, config)
	if err != nil {
		return "", err
	}

	// Track usage
	if resp.UsageMetadata != nil {
		c.usage.TotalCalls++
		c.usage.TotalInputTokens += int(resp.UsageMetadata.PromptTokenCount)
		c.usage.TotalOutputTokens += int(resp.UsageMetadata.CandidatesTokenCount)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
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
