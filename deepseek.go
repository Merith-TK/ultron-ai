package main

import (
	"context"
	"fmt"

	deepseek "github.com/cohesion-org/deepseek-go"
)

type DeepSeekClient struct {
	client *deepseek.Client
	model  string
}

func NewDeepSeekClient(apiKey, model string) *DeepSeekClient {
	return &DeepSeekClient{
		client: deepseek.NewClient(apiKey),
		model:  model,
	}
}

func (d *DeepSeekClient) CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	resp, err := d.client.CreateChatCompletion(ctx, &deepseek.ChatCompletionRequest{
		Model:    d.model,
		Messages: toDeepSeekMessages(req.Messages),
	})
	if err != nil {
		return nil, fmt.Errorf("DeepSeek API error: %w", err)
	}

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
	var result []deepseek.ChatCompletionMessage
	for _, msg := range messages {
		result = append(result, deepseek.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return result
}
