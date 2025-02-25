package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/Merith-TK/utils/debug"
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
	flag.Parse()

	if err := loadConfig(); err != nil {
		fmt.Println("Failed to load config:", err)
		os.Exit(1)
	}

	if !debug.GetDebug() {
		debug.SetDebug(cfg.Debug)
		debug.Print("Debug mode enabled.")
	}

	debug.SetTitle("ULTRON-AI")
	debug.Print("Starting Ultron-AI...")

	// Initialize the appropriate client based on the backend
	switch cfg.AIProvider.Backend {
	case "deepseek":
		debug.Print("Initializing DeepSeek client...")
		client = NewDeepSeekClient(cfg.AIProvider.DeepSeek.Key, cfg.AIProvider.DeepSeek.Model)
	case "openai":
		debug.Print("Initializing OpenAI client...")
		client = NewOpenAIClient(cfg.AIProvider.OpenAI.Key, cfg.AIProvider.OpenAI.Model)
	case "custom":
		debug.Print("Initializing custom AI client...")
		client = NewCustomAIClient(cfg.AIProvider.Custom.Key, cfg.AIProvider.Custom.Model, cfg.AIProvider.Custom.URL)
	default:
		fmt.Println("Unsupported backend:", cfg.AIProvider.Backend)
		os.Exit(1)
	}

	// Initialize conversation history
	conversationHistory = append(conversationHistory, ChatCompletionMessage{
		Role:    "system",
		Content: cfg.AIProvider.Prompt, // Use the prompt from the config
	})
	debug.Print("Conversation history initialized with system prompt.")

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Ultron-AI ready. Enter a command:")

	for {
		fmt.Print("$ ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		debug.Print("User input:", input)

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
		debug.Print("Turtle state retrieved:", turtleState)

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
	debug.Print("Fetching turtle state from:", cfg.Ultron.APIUrl+"/api/turtle/"+cfg.Ultron.TurtleID)
	resp, err := http.Get(cfg.Ultron.APIUrl + "/api/turtle/" + cfg.Ultron.TurtleID)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	debug.Print("Turtle state response:", string(body))
	return string(body), nil
}

func processCommand(userInput, turtleState string) (string, error) {
	// Send both turtle state and user input to OpenAI
	conversationHistory = append(conversationHistory, ChatCompletionMessage{
		Role:    "user",
		Content: fmt.Sprintf("Turtle State: %s\nUser Command: %s", turtleState, userInput),
	})
	debug.Print("Updated conversation history with user input.")

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
	debug.Print("Updated conversation history with AI response.")

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
	debug.SetTitle("TURTLE")
	var parsedCommands []string
	command = strings.TrimSuffix(command, "```")
	command = strings.TrimPrefix(command, "```json")
	command = strings.TrimPrefix(command, "```")
	command = strings.TrimSpace(command)

	// Validate the command is a valid JSON array
	if err := json.Unmarshal([]byte(command), &parsedCommands); err != nil {
		return fmt.Errorf("invalid command format: %v", err)
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
	debug.Print("Sending command to turtle:", string(requestBody))
	debug.Print("Turtle API URL:", cfg.Ultron.APIUrl+"/api/turtle/"+cfg.Ultron.TurtleID)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	debug.Print("Response from turtle API:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	debug.Print("Response body:", string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send command to turtle: %s", resp.Status)
	}

	return nil
}
