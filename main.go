package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model    string                  `json:"model"`
	Messages []ChatCompletionMessage `json:"messages"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message ChatCompletionMessage `json:"message"`
	} `json:"choices"`
}

type AIProvider interface {
	CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error)
}

var (
	client              AIProvider
	conversationHistory []ChatCompletionMessage
)

func main() {
	if err := loadConfig(); err != nil {
		fmt.Println("Failed to load config:", err)
		os.Exit(1)
	}

	// Initialize the appropriate client based on the backend
	switch cfg.AIProvider.Backend {
	case "deepseek":
		client = NewDeepSeekClient(cfg.AIProvider.DeepSeek.Key, cfg.AIProvider.DeepSeek.Model)
	case "openai":
		client = NewOpenAIClient(cfg.AIProvider.OpenAI.Key, cfg.AIProvider.OpenAI.Model)
	default:
		fmt.Println("Unsupported backend:", cfg.AIProvider.Backend)
		os.Exit(1)
	}

	// Initialize conversation history
	conversationHistory = append(conversationHistory, ChatCompletionMessage{
		Role:    "system",
		Content: "You are a helpful assistant.",
	})

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Ultron-AI ready. Enter a command:")

	for {
		fmt.Print("$ ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}
		if strings.ToLower(input) == "exit" {
			fmt.Println("Exiting Ultron-AI.")
			break
		}

		// Get current turtle state
		turtleState, err := getTurtleState()
		if err != nil {
			fmt.Println("Error getting turtle state:", err)
			continue
		}

		// Process user input with AI
		response, err := processCommand(input, turtleState)
		if err != nil {
			fmt.Println("Error processing command:", err)
			continue
		}

		fmt.Println("AI Response:", response)

		// Send command to the turtle
		err = sendToTurtle(response)
		if err != nil {
			fmt.Println("Error sending command to turtle:", err)
		}
	}
}

func getTurtleState() (string, error) {
	resp, err := http.Get(cfg.Ultron.APIUrl + "/api/turtle/" + cfg.Ultron.TurtleID)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func processCommand(userInput, turtleState string) (string, error) {
	// Send both turtle state and user input to OpenAI
	conversationHistory = append(conversationHistory, ChatCompletionMessage{
		Role:    "user",
		Content: fmt.Sprintf("Turtle State: %s\nUser Command: %s", turtleState, userInput),
	})

	resp, err := client.CreateChatCompletion(context.Background(), &ChatCompletionRequest{
		Model:    cfg.AIProvider.OpenAI.Model, // Use the model from the config
		Messages: conversationHistory,
	})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	aiResponse := cleanAIResponse(resp.Choices[0].Message.Content)
	conversationHistory = append(conversationHistory, ChatCompletionMessage{
		Role:    "assistant",
		Content: aiResponse,
	})

	return aiResponse, nil
}

func cleanAIResponse(response string) string {
	// Remove markdown-style JSON formatting
	re := regexp.MustCompile("(?s)```json\\n(.*?)\\n```")
	matches := re.FindStringSubmatch(response)
	if len(matches) > 1 {
		response = matches[1] // Extract the actual JSON content
	}

	// Trim whitespace
	return strings.TrimSpace(response)
}

func sendToTurtle(command string) error {
	var parsedCommands []string
	command = strings.TrimSuffix(command, "```")
	command = strings.TrimPrefix(command, "```json")
	command = strings.TrimPrefix(command, "```")
	command = strings.TrimSpace(command)

	if err := json.Unmarshal([]byte(command), &parsedCommands); err != nil {
		parsedCommands = []string{command} // If not JSON, assume single command
	}

	// Ensure API receives an array
	requestBody, err := json.Marshal(parsedCommands)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", cfg.Ultron.APIUrl+"/api/turtle/"+cfg.Ultron.TurtleID, strings.NewReader(string(requestBody)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send command to turtle: %s", resp.Status)
	}

	return nil
}
