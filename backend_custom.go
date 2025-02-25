package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Merith-TK/utils/debug"
)

// CustomBackendClient is a client for interacting with a custom backend.
type CustomBackendClient struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewCustomBackendClient initializes a new client for the custom backend.
func NewCustomAIClient(apiKey, model, baseURL string) *CustomBackendClient {
	debug.Print("Initializing Custom Backend client with model:", model)
	return &CustomBackendClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateChatCompletion sends a chat completion request to the custom backend.
func (c *CustomBackendClient) CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	debug.Print("Creating chat completion with Custom Backend API...")
	debug.Print("Request model:", c.model)
	debug.Print("Request messages:", req.Messages)

	// Prepare the request payload
	payload := ChatCompletionRequest{
		Model:    c.model,
		Messages: req.Messages,
	}

	// Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		debug.Print("Error marshaling request payload:", err)
		return nil, fmt.Errorf("error marshaling request payload: %w", err)
	}

	// Create a new HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewBuffer(jsonPayload))
	if err != nil {
		debug.Print("Error creating HTTP request:", err)
		return nil, fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Send the request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		debug.Print("Custom Backend API error:", err)
		return nil, fmt.Errorf("custom backend API error: %w", err)
	}
	defer resp.Body.Close()

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		debug.Print("Custom Backend API returned non-200 status code:", resp.StatusCode)
		return nil, fmt.Errorf("custom Backend API returned non-200 status code: %d", resp.StatusCode)
	}

	// Decode the response
	var chatResponse ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResponse); err != nil {
		debug.Print("Error decoding Custom Backend API response:", err)
		return nil, fmt.Errorf("error decoding Custom Backend API response: %w", err)
	}

	debug.Print("Custom Backend API response received successfully.")
	debug.Print("Response content:", chatResponse.Choices[0].Message.Content)

	return &chatResponse, nil
}
