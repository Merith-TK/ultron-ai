// main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	StatePollInterval   = 3 * time.Second
	CommandCooldown     = 1 * time.Second
	MaxConsecutiveFails = 5
)

type Agent struct {
	Client         AIProvider
	StateAPI       string
	TurtleID       string
	Conversation   []ChatCompletionMessage
	LastCommand    string
	CycleCount     int
	ConsecutiveFails int
}

func main() {
	if err := initialize(); err != nil {
		logFatal("Initialization failed:", err)
	}

	agent := &Agent{
		Client:   client,
		StateAPI: fmt.Sprintf("%s/api/turtle/%s", cfg.Ultron.APIUrl, cfg.Ultron.TurtleID),
		TurtleID: cfg.Ultron.TurtleID,
		Conversation: []ChatCompletionMessage{
			{Role: "system", Content: cfg.AIProvider.Prompt},
		},
	}

	if initialTask := loadInitialTask(); initialTask != "" {
		agent.addUserMessage("Initial Task: " + initialTask)
	}

	agent.runAutonomousLoop()
}

func (a *Agent) runAutonomousLoop() {
	ticker := time.NewTicker(StatePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.CycleCount++
			logInfo("Starting operation cycle", a.CycleCount)

			if a.ConsecutiveFails >= MaxConsecutiveFails {
				logError("Critical failure threshold reached, aborting")
				return
			}

			state, err := a.getTurtleState()
			if err != nil {
				logError("State fetch failed:", err)
				a.ConsecutiveFails++
				continue
			}

			command, err := a.generateCommand(state)
			if err != nil {
				logError("Command generation failed:", err)
				a.ConsecutiveFails++
				continue
			}

			if isTaskComplete(command) {
				logInfo("Task completed successfully")
				return
			}

			if err := a.executeCommand(command); err != nil {
				logError("Command execution failed:", err)
				a.addSystemMessage("Command failed: " + err.Error())
				a.ConsecutiveFails++
				continue
			}

			a.ConsecutiveFails = 0
			time.Sleep(CommandCooldown)
		}
	}
}

func (a *Agent) getTurtleState() (string, error) {
	resp, err := http.Get(a.StateAPI)
	if err != nil {
		return "", fmt.Errorf("API unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid status: %s", resp.Status)
	}

	var state struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return "", fmt.Errorf("invalid state format: %w", err)
	}

	stateJson, _ := json.MarshalIndent(state.Data, "", "  ")
	return string(stateJson), nil
}

func (a *Agent) generateCommand(state string) (string, error) {
	a.addUserMessage(fmt.Sprintf(`Current State:
%s
Previous Command Result: %s`, state, a.LastCommand))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := a.Client.CreateChatCompletion(ctx, &ChatCompletionRequest{
		Model:    cfg.AIProvider.OpenAI.Model,
		Messages: a.Conversation,
	})
	if err != nil {
		return "", fmt.Errorf("AI provider error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty AI response")
	}

	command := cleanAIResponse(resp.Choices[0].Message.Content)
	a.addAssistantMessage(command)
	return command, nil
}

func (a *Agent) executeCommand(command string) error {
	reqBody := struct {
		Commands []string `json:"commands"`
	}{[]string{command}}

	req, _ := http.NewRequest("POST", a.StateAPI, jsonMarshal(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	a.LastCommand = fmt.Sprintf("Success: %v", result.Success)
	if !result.Success {
		a.LastCommand += fmt.Sprintf(" | Error: %s", result.Error)
		return fmt.Errorf("command execution failed: %s", result.Error)
	}
	return nil
}

// Helper methods
func (a *Agent) addUserMessage(content string) {
	a.Conversation = append(a.Conversation, ChatCompletionMessage{
		Role:    "user",
		Content: content,
	})
}

func (a *Agent) addAssistantMessage(content string) {
	a.Conversation = append(a.Conversation, ChatCompletionMessage{
		Role:    "assistant",
		Content: content,
	})
}

func (a *Agent) addSystemMessage(content string) {
	a.Conversation = append(a.Conversation, ChatCompletionMessage{
		Role:    "system",
		Content: content,
	})
}

func cleanAIResponse(response string) string {
	response = regexp.MustCompile(`(?s)```lua(.*?)````).ReplaceAllString(response, "$1")
	return strings.TrimSpace(response)
}

func isTaskComplete(command string) bool {
	return strings.Contains(strings.ToLower(command), "task complete")
}

func jsonMarshal(v interface{}) *strings.Reader {
	b, _ := json.Marshal(v)
	return strings.NewReader(string(b))
}