package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/sashabaranov/go-openai"
)

const (
	lmstudioModel = "qwen2.5-7b-instruct"
)

func TestLMStudio_Chat(t *testing.T) {
	config := openai.DefaultConfig("dummy-key")
	config.BaseURL = "http://localhost:1234/v1"
	client := openai.NewClientWithConfig(config)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: lmstudioModel,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Hello! 用一句话介绍 Go 语言",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("请求失败：%v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("Empty choices")
	}

	if resp.Choices[0].Message.Content == "" {
		t.Error("Empty response content")
	}

	t.Logf("Response: %s", resp.Choices[0].Message.Content)
}

// TestLMStudio_ImageUnderstanding 测试LM Studio的图像理解功能
// 需要在LM Studio中加载支持视觉的Qwen模型(如qwen2.5-vl-7b-instruct)
func TestLMStudio_ImageUnderstanding(t *testing.T) {
	// 模型名称 - 使用LM Studio中可用的视觉模型
	model := "qwen/qwen3-vl-8b"

	// 图片路径
	imagePath := "../screenshot_1772360863272.png"

	// 读取图片文件
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("读取图片文件失败: %v", err)
	}

	// 将图片编码为base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	config := openai.DefaultConfig("dummy-key")
	config.BaseURL = "http://localhost:1234/v1"
	client := openai.NewClientWithConfig(config)

	ctx := context.Background()

	// 发送图片给模型，让它识别"添加配置"文本的位置
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleUser,
					MultiContent: []openai.ChatMessagePart{
						{
							Type: "text",
							Text: "请仔细查看这张图片，找出\"添加配置\"这个文本在图片中的位置（坐标或区域）。",
						},
						{
							Type: "image_url",
							ImageURL: &openai.ChatMessageImageURL{
								URL: fmt.Sprintf("data:image/png;base64,%s", base64Image),
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("Empty choices")
	}

	t.Logf("模型识别结果: %s", resp.Choices[0].Message.Content)
}

func TestLMStudio_Tools(t *testing.T) {
	config := openai.DefaultConfig("dummy-key")
	config.BaseURL = "http://localhost:1234/v1"
	client := openai.NewClientWithConfig(config)

	weatherTool := openai.Tool{
		Type: "function",
		Function: &openai.FunctionDefinition{
			Name:        "get_weather",
			Description: "获取指定城市的天气信息",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"city": map[string]interface{}{
						"type":        "string",
						"description": "城市名称，如北京、上海",
					},
				},
				"required": []string{"city"},
			},
		},
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "北京今天天气怎么样？",
		},
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    lmstudioModel,
			Messages: messages,
			Tools:    []openai.Tool{weatherTool},
		},
	)
	if err != nil {
		t.Fatalf("请求失败：%v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("Empty choices")
	}

	choice := resp.Choices[0]

	if len(choice.Message.ToolCalls) > 0 {
		t.Logf("Tool calls detected: %d", len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			t.Logf("  - Tool: %s, Arguments: %s", tc.Function.Name, tc.Function.Arguments)
		}

		toolCall := choice.Message.ToolCalls[0]
		args := make(map[string]interface{})
		json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

		messages = append(messages, choice.Message)
		messages = append(messages, openai.ChatCompletionMessage{
			Role:         openai.ChatMessageRoleTool,
			ToolCallID:   toolCall.ID,
			Content:      "北京今天天气晴朗，温度 15-25°C，适宜外出。",
			Name:         toolCall.Function.Name,
			FunctionCall: nil,
		})

		resp2, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:    lmstudioModel,
				Messages: messages,
				Tools:    []openai.Tool{weatherTool},
			},
		)
		if err != nil {
			t.Fatalf("第二次请求失败：%v", err)
		}

		t.Logf("Final response: %s", resp2.Choices[0].Message.Content)
	} else {
		t.Logf("No tool calls, direct response: %s", choice.Message.Content)
	}
}
