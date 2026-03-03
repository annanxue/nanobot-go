package providers

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
)

func TestOpenAIImageUnderstanding(t *testing.T) {
	godotenv.Load("../.env")

	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping test")
	}

	imagePath := os.Getenv("TEST_IMAGE_PATH")
	if imagePath == "" {
		t.Skip("TEST_IMAGE_PATH not set, skipping test")
	}

	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("Failed to read image file: %v", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	client := openai.NewClientWithConfig(cfg)

	ctx := context.Background()

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "qwen-omni-turbo",
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: "text",
						Text: "What is in this image? Please describe it in detail.",
					},
					{
						Type: "image_url",
						ImageURL: &openai.ChatMessageImageURL{
							URL: fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
						},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Failed to get response from OpenAI: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("No response choices returned")
	}

	fmt.Printf("Image Analysis Result:\n%s\n", resp.Choices[0].Message.Content)
}

func TestOpenAIProviderImageUnderstanding(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping test")
	}

	imagePath := os.Getenv("TEST_IMAGE_PATH")
	if imagePath == "" {
		t.Skip("TEST_IMAGE_PATH not set, skipping test")
	}

	provider := NewOpenAIProvider(apiKey, "", "gpt-4o", nil, "openai")

	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("Failed to read image file: %v", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	messages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleUser,
			MultiContent: []openai.ChatMessagePart{
				{
					Type: "text",
					Text: "What is in this image? Please describe it in detail.",
				},
				{
					Type: "image_url",
					ImageURL: &openai.ChatMessageImageURL{
						URL: fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
					},
				},
			},
		},
	}

	ctx := context.Background()

	resp, err := provider.Chat(ctx, messages, nil, "gpt-4o", 4096, 0.7)
	if err != nil {
		t.Fatalf("Failed to get response from provider: %v", err)
	}

	fmt.Printf("Image Analysis Result:\n%s\n", resp.Content)
}
