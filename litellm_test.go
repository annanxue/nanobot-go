package main

import (
	"context"
	"testing"

	"github.com/voocel/litellm"
)

func TestLiteLLM_Chat(t *testing.T) {
	client, err := litellm.NewWithProvider("deepseek", litellm.ProviderConfig{
		APIKey: "sk-fa177959dc4343a785e4c4e2ac687bce",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	resp, err := client.Chat(context.Background(), &litellm.Request{
		Model:    "deepseek-",
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
