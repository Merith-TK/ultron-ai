package main

import (
	"context"
	"fmt"

	"github.com/Merith-TK/utils/debug"
	deepseek "github.com/cohesion-org/deepseek-go"
)

type DeepSeekClient struct {
	client *deepseek.Client
	model  string
}

func NewDeepSeekClient(apiKey, model string) *DeepSeekClient {
	debug.Print("Initializing DeepSeek client with model:", model)
	return &DeepSeekClient{
		client: deepseek.NewClient(apiKey),
		model:  model,
	}
}

func (d *DeepSeekClient) CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	debug.Print("Creating chat completion with DeepSeek API...")
	debug.Print("Request model:", d.model)
	debug.Print("Request messages:", req.Messages)

	resp, err := d.client.CreateChatCompletion(ctx, &deepseek.ChatCompletionRequest{
		Model:    d.model,
		Messages: toDeepSeekMessages(req.Messages),
	})
	if err != nil {
		debug.Print("DeepSeek API error:", err)
		return nil, fmt.Errorf("DeepSeek API error: %w", err)
	}

	debug.Print("DeepSeek API response received successfully.")
	debug.Print("Response content:", resp.Choices[0].Message.Content)

	return &ChatCompletionResponse{
		Choices: []struct {
			Message ChatCompletionMessage `json:"message"`
		}{
			{
				Message: ChatCompletionMessage{
					Role:    resp.Choices[0].Message.Role,
					Content: resp.Choices[0].Message.Content,
				},
			},
		},
	}, nil
}

func toDeepSeekMessages(messages []ChatCompletionMessage) []deepseek.ChatCompletionMessage {
	debug.Print("Converting messages to DeepSeek format...")
	var result []deepseek.ChatCompletionMessage
	for _, msg := range messages {
		result = append(result, deepseek.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	debug.Print("Messages converted successfully.")
	return result
}
