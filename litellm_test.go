package main

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/voocel/litellm"
)

func TestLiteLLM_Chat(t *testing.T) {
	godotenv.Load()

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		t.Skip("DEEPSEEK_API_KEY not set in .env file")
	}

	client, err := litellm.NewWithProvider("deepseek", litellm.ProviderConfig{
		APIKey: apiKey,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	resp, err := client.Chat(context.Background(), &litellm.Request{
		Model:    "deepseek-reasoner",
		Messages: []litellm.Message{{Role: "user", Content: "Explain AI in one sentence."}},
	})
	if err != nil {
		t.Fatalf("Failed to chat: %v", err)
	}

	if resp.Content == "" {
		t.Error("Empty response content")
	}

	t.Logf("Response: %s", resp.Content)
}
