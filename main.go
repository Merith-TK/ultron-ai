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
	"time"

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
	logFile             *os.File
	historyFile         *os.File
)

func main() {
	flag.Parse()

	// Create logs directory if it doesn't exist
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		err := os.Mkdir("logs", 0755)
		if err != nil {
			fmt.Println("Failed to create logs directory:", err)
			os.Exit(1)
		}
	}

	// Initialize logging
	var err error
	logFile, err = os.OpenFile("logs/runtime.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open log file:", err)
		os.Exit(1)
	}
	defer logFile.Close()

	// Initialize history logging
	historyFile, err = os.OpenFile("logs/history.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open history file:", err)
		os.Exit(1)
	}

	log("Starting Ultron-AI...")

	if err := loadConfig(); err != nil {
		log("Failed to load config:", err)
		os.Exit(1)
	}

	if !debug.GetDebug() {
		debug.SetDebug(cfg.Debug)
		log("Debug mode enabled.")
	}

	debug.SetTitle("ULTRON-AI")

	// Initialize the appropriate client based on the backend
	log("Attempting to initialize AI provider:", cfg.AIProvider.Backend)
	switch cfg.AIProvider.Backend {
	case "deepseek":
		client = NewDeepSeekClient(cfg.AIProvider.DeepSeek.Key, cfg.AIProvider.DeepSeek.Model)
	case "openai":
		client = NewOpenAIClient(cfg.AIProvider.OpenAI.Key, cfg.AIProvider.OpenAI.Model)
	case "custom":
		client = NewCustomAIClient(cfg.AIProvider.Custom.Key, cfg.AIProvider.Custom.Model, cfg.AIProvider.Custom.URL)
	default:
		os.Exit(1)
	}

	// Load conversation history if it exists
	if err := loadConversationHistory(); err != nil {
		log("Failed to load conversation history:", err)
	}

	// Initialize conversation history with system prompt if empty
	if len(conversationHistory) == 0 {
		conversationHistory = append(conversationHistory, ChatCompletionMessage{
			Role:    "system",
			Content: cfg.AIProvider.Prompt, // Use the prompt from the config
		})
		log("Conversation history initialized with system prompt.")
	} else {
		log("Resuming from existing conversation history.")
	}

	// Load initial task from init-task.txt
	initialTask, err := loadInitialTask("init-task.txt")
	if err != nil {
		log("Error loading initial task:", err)
	} else if initialTask != "" {
		log("Initial task loaded:", initialTask)
		conversationHistory = append(conversationHistory, ChatCompletionMessage{
			Role:    "user",
			Content: initialTask,
		})
	}

	// Start the automated interaction loop
	for {
		// Get current turtle state
		turtleState, err := getTurtleState()
		if err != nil {
			log("Error getting turtle state:", err)
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}
		log("Turtle state retrieved:", turtleState)

		// Process the state with AI
		response, err := processCommand("", turtleState)
		if err != nil {
			log("Error processing command:", err)
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}
		log("AI Response:", response)

		// Check if the task is complete
		if isTaskComplete(response) {
			log("Task completed successfully.")
			break
		}

		// Send command to the turtle
		err = sendToTurtle(response)
		if err != nil {
			log("Error sending command to turtle:", err)
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}

		// Save conversation history after each interaction
		if err := saveConversationHistory("conversation_history.json"); err != nil {
			log("Failed to save conversation history:", err)
		}

		// Wait for the turtle to execute the command
		time.Sleep(5 * time.Second)
	}

	// Clear conversation history after task completion
	if err := clearConversationHistory("conversation_history.json"); err != nil {
		log("Failed to clear conversation history:", err)
	}
}

func loadInitialTask(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var task strings.Builder
	for scanner.Scan() {
		task.WriteString(scanner.Text())
		task.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.TrimSpace(task.String()), nil
}

func loadConversationHistory() error {
	decoder := json.NewDecoder(historyFile)
	if err := decoder.Decode(&conversationHistory); err != nil {
		return err
	}

	log("Loaded conversation history from:", historyFile.Name())
	return nil
}

func saveConversationHistory(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(conversationHistory); err != nil {
		return err
	}

	log("Saved conversation history to:", filename)
	return nil
}

func clearConversationHistory(filename string) error {
	if err := os.Remove(filename); err != nil {
		return err
	}

	log("Cleared conversation history.")
	return nil
}

func getTurtleState() (string, error) {
	log("Fetching turtle state from:", cfg.Ultron.APIUrl+"/api/turtle/"+cfg.Ultron.TurtleID)
	resp, err := http.Get(cfg.Ultron.APIUrl + "/api/turtle/" + cfg.Ultron.TurtleID)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	log("Turtle state response:", string(body))
	return string(body), nil
}

func processCommand(userInput, turtleState string) (string, error) {
	// Send both turtle state and user input to AI
	conversationHistory = append(conversationHistory, ChatCompletionMessage{
		Role:    "user",
		Content: fmt.Sprintf("Turtle State: %s\nUser Command: %s", turtleState, userInput),
	})
	log("Updated conversation history with user input.")

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
	log("Updated conversation history with AI response.")

	return aiResponse, nil
}

func cleanAIResponse(response string) string {
	// Extract Lua code from markdown code blocks if present
	re := regexp.MustCompile("(?s)```lua\n(.*?)\n```")
	matches := re.FindStringSubmatch(response)
	if len(matches) > 1 {
		response = matches[1]
	}

	// Remove any remaining markdown or non-Lua code
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	return strings.TrimSpace(response)
}

func sendToTurtle(luaCode string) error {
	log("Sending Lua code to turtle:", luaCode)

	// Wrap the Lua code in a JSON array with single element
	commands := []string{luaCode}
	requestBody, err := json.Marshal(commands)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %v", err)
	}

	req, err := http.NewRequest("POST", cfg.Ultron.APIUrl+"/api/turtle/"+cfg.Ultron.TurtleID,
		strings.NewReader(string(requestBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()
	log("Response from turtle API:", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log("Response body:", string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send command to turtle: %s", resp.Status)
	}

	return nil
}

func isTaskComplete(response string) bool {
	// Implement logic to determine if the task is complete
	// For example, check if the AI response contains a completion message
	return strings.Contains(strings.ToLower(response), "task complete")
}

func log(args ...interface{}) {
	message := fmt.Sprintln(args...)
	fmt.Print(message)
	if logFile != nil {
		logFile.WriteString(message)
	}
}
