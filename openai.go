package main

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
	model  string
}

func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	return &OpenAIClient{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

func (o *OpenAIClient) CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	resp, err := o.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    o.model,
		Messages: toOpenAIMessages(req.Messages),
	})
	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
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

func toOpenAIMessages(messages []ChatCompletionMessage) []openai.ChatCompletionMessage {
	var result []openai.ChatCompletionMessage
	for _, msg := range messages {
		result = append(result, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return result
}
