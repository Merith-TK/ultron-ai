package main

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/Merith-TK/utils/debug"
	"github.com/ncruces/zenity"
	"github.com/pelletier/go-toml/v2"
)

//go:embed prompt.md
var defaultPrompt embed.FS

type Config struct {
	Debug      bool             `toml:"debug,omitempty"` // Enable debug mode
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
	debug.Print("Loading configuration...")
	data, err := os.ReadFile("config.toml")
	if err != nil {
		if os.IsNotExist(err) {
			debug.Print("Config file not found. Creating default config...")
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
				debug.Print("Failed to marshal default config:", err)
				return fmt.Errorf("failed to marshal default config: %w", err)
			}

			// Write the default config to file
			if err := os.WriteFile("config.toml", data, 0644); err != nil {
				debug.Print("Failed to write default config file:", err)
				return fmt.Errorf("failed to write default config file: %w", err)
			}

			debug.Print("Default config file created successfully.")
			return nil
		}
		debug.Print("Failed to read config file:", err)
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		debug.Print("Failed to parse config:", err)
		return fmt.Errorf("failed to parse config: %w", err)
	}

	debug.Print("Config file loaded successfully.")

	// If default values are used, display a popup using zenity
	if cfg.AIProvider.OpenAI.Key == "default-key" && cfg.AIProvider.OpenAI.Model == "default-model" {
		debug.Print("Default OpenAI values detected. Showing warning popup.")
		zenity.Warning("Default values are used for OpenAI API key and model. Please update them in the config file.")
		os.Exit(1)
	}

	// Handle the prompt file
	if err := handlePromptFile(); err != nil {
		debug.Print("Failed to handle prompt file:", err)
		return fmt.Errorf("failed to handle prompt file: %w", err)
	}
	debug.Print("Prompt file handled successfully.")

	// Clean the turtle URL
	cfg.Ultron.APIUrl = cleanTurtleURL(cfg.Ultron.APIUrl)
	debug.Print("Turtle URL cleaned successfully.")

	return nil
}

func handlePromptFile() error {
	// Default to ./prompt.md if no prompt file is specified
	if cfg.PromptFile == "" {
		cfg.PromptFile = "./prompt.md"
		debug.Print("No prompt file specified. Defaulting to:", cfg.PromptFile)
	}

	// Check if the prompt file exists
	if _, err := os.Stat(cfg.PromptFile); os.IsNotExist(err) {
		// If the file doesn't exist, create it with the embedded prompt
		fmt.Printf("Prompt file '%s' not found. Creating it with the default prompt.\n", cfg.PromptFile)

		// Read the embedded default prompt
		data, err := defaultPrompt.ReadFile("prompt.md")
		if err != nil {
			debug.Print("Failed to read embedded prompt:", err)
			return fmt.Errorf("failed to read embedded prompt: %w", err)
		}

		// Write the embedded prompt to the file
		if err := os.WriteFile(cfg.PromptFile, data, 0644); err != nil {
			debug.Print("Failed to write prompt file:", err)
			return fmt.Errorf("failed to write prompt file: %w", err)
		}
		debug.Print("Prompt file created successfully.")
	} else if err != nil {
		debug.Print("Failed to check prompt file:", err)
		return fmt.Errorf("failed to check prompt file: %w", err)
	}

	// Load the prompt content into Config.Prompt
	promptData, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		debug.Print("Failed to read prompt file:", err)
		return fmt.Errorf("failed to read prompt file: %w", err)
	}
	cfg.AIProvider.Prompt = string(promptData)
	debug.Print("Prompt content loaded into config.")

	return nil
}

func cleanTurtleURL(url string) string {
	debug.Print("Cleaning turtle URL...")
	// Default URL if none is provided
	if url == "" {
		debug.Print("No URL provided. Using default URL.")
		return fmt.Sprintf("http://localhost:3300/api/turtle/%s", cfg.Ultron.TurtleID)
	}

	// Ensure the URL does not end with a slash
	url = strings.TrimSuffix(url, "/")
	debug.Print("URL trimmed of trailing slash.")

	debug.Print("Cleaned turtle URL:", url)
	return url
}
