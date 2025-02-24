package main

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/ncruces/zenity"
	"github.com/pelletier/go-toml/v2"
)

//go:embed prompt.md
var defaultPrompt embed.FS

type Config struct {
	AIProvider AIProviderConfig `toml:"ai_provider"`
	Ultron     UltronConfig     `toml:"ultron"`
	PromptFile string           `toml:"prompt_file,omitempty"` // Path to the prompt file
}

type UltronConfig struct {
	APIUrl   string `toml:"api_url"`
	TurtleID string `toml:"turtle_id"`
}

type AIProviderConfig struct {
	Backend  string          `toml:"backend"`
	Prompt   string          `toml:"prompt,omitempty"` // The actual prompt content
	DeepSeek CommonAPIConfig `toml:"deepseek"`
	OpenAI   CommonAPIConfig `toml:"openai"`
	Custom   CommonAPIConfig `toml:"custom"`
}

type CommonAPIConfig struct {
	URL   string `toml:"url,omitempty"`
	Key   string `toml:"key"`
	Model string `toml:"model"`
}

var cfg Config

func loadConfig() error {
	data, err := os.ReadFile("config.toml")
	if err != nil {
		if os.IsNotExist(err) {
			// Populate with default values
			cfg = Config{
				AIProvider: AIProviderConfig{
					Backend: "openai",
					OpenAI: CommonAPIConfig{
						Key:   "default-key",
						Model: "default-model",
					},
				},
				Ultron: UltronConfig{
					APIUrl:   "http://localhost:3300/",
					TurtleID: "0",
				},
				PromptFile: "./prompt.md", // Default prompt file path
			}

			// Marshal the default config to TOML
			data, err := toml.Marshal(&cfg)
			if err != nil {
				return fmt.Errorf("failed to marshal default config: %w", err)
			}

			// Write the default config to file
			if err := os.WriteFile("config.toml", data, 0644); err != nil {
				return fmt.Errorf("failed to write default config file: %w", err)
			}

			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// If default values are used, display a popup using zenity
	if cfg.AIProvider.OpenAI.Key == "default-key" && cfg.AIProvider.OpenAI.Model == "default-model" {
		zenity.Warning("Default values are used for OpenAI API key and model. Please update them in the config file.")
		os.Exit(1)
	}

	// Handle the prompt file
	if err := handlePromptFile(); err != nil {
		return fmt.Errorf("failed to handle prompt file: %w", err)
	}

	return nil
}

func handlePromptFile() error {
	// Default to ./prompt.md if no prompt file is specified
	if cfg.PromptFile == "" {
		cfg.PromptFile = "./prompt.md"
	}

	// Check if the prompt file exists
	if _, err := os.Stat(cfg.PromptFile); os.IsNotExist(err) {
		// If the file doesn't exist, create it with the embedded prompt
		fmt.Printf("Prompt file '%s' not found. Creating it with the default prompt.\n", cfg.PromptFile)

		// Read the embedded default prompt
		data, err := defaultPrompt.ReadFile("prompt.md")
		if err != nil {
			return fmt.Errorf("failed to read embedded prompt: %w", err)
		}

		// Write the embedded prompt to the file
		if err := os.WriteFile(cfg.PromptFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write prompt file: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check prompt file: %w", err)
	}

	// Load the prompt content into Config.Prompt
	promptData, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		return fmt.Errorf("failed to read prompt file: %w", err)
	}
	cfg.AIProvider.Prompt = string(promptData)

	return nil
}

func cleanTurtleURL(url string) string {
	// Default URL if none is provided
	if url == "" {
		return fmt.Sprintf("http://localhost:3300/api/turtle/%s", cfg.Ultron.TurtleID)
	}

	// Ensure the URL does not end with a slash
	url = strings.TrimSuffix(url, "/")

	// If the URL does not contain "/api/turtle/", append it along with the TurtleID
	if !strings.Contains(url, "/api/turtle/") {
		url = fmt.Sprintf("%s/api/turtle/%s", url, cfg.Ultron.TurtleID)
	}

	return url
}
