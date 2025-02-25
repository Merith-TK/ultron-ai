package main

import (
	"context"
	"fmt"

	"github.com/Merith-TK/utils/debug"
	openai "github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
	model  string
}

func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	debug.Print("Initializing OpenAI client with model:", model)
	return &OpenAIClient{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

func (o *OpenAIClient) CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	debug.Print("Creating chat completion with OpenAI API...")
	debug.Print("Request model:", o.model)
	debug.Print("Request messages:", req.Messages)

	resp, err := o.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    o.model,
		Messages: toOpenAIMessages(req.Messages),
	})
	if err != nil {
		debug.Print("OpenAI API error:", err)
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	debug.Print("OpenAI API response received successfully.")
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

func toOpenAIMessages(messages []ChatCompletionMessage) []openai.ChatCompletionMessage {
	debug.Print("Converting messages to OpenAI format...")
	var result []openai.ChatCompletionMessage
	for _, msg := range messages {
		result = append(result, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	debug.Print("Messages converted successfully.")
	return result
}
