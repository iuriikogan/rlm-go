package client

import (
	"os"
	"testing"
)

func TestNewGeminiClient(t *testing.T) {
	// Test error when API key is missing
	os.Setenv("GEMINI_API_KEY", "")
	_, err := NewGeminiClient("", "")
	if err == nil {
		t.Error("Expected error when GEMINI_API_KEY is missing")
	}

	// Test with explicit key
	c, err := NewGeminiClient("dummy-key", "gemini-model")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if c.modelName != "gemini-model" {
		t.Errorf("Expected model gemini-model, got %s", c.modelName)
	}
}
