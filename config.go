package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	AIProvider AIProviderConfig `toml:"ai_provider"`
	Ultron     UltronConfig     `toml:"ultron"`
}

type UltronConfig struct {
	APIUrl   string `toml:"api_url"`
	TurtleID string `toml:"turtle_id"`
}

type AIProviderConfig struct {
	Backend  string         `toml:"backend"`
	DeepSeek DeepSeekConfig `toml:"deepseek,omitempty"`
	OpenAI   OpenAIConfig   `toml:"openai,omitempty"`
	Custom   CustomConfig   `toml:"custom,omitempty"`
}

type DeepSeekConfig struct {
	Key   string `toml:"key"`
	Model string `toml:"model"`
}

type OpenAIConfig struct {
	Key   string `toml:"key"`
	Model string `toml:"model"`
}

type CustomConfig struct {
	URL   string `toml:"url"`
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
					OpenAI: OpenAIConfig{
						Key:   "default-key",
						Model: "default-model",
					},
				},
				Ultron: UltronConfig{
					APIUrl:   "http://localhost:3300/",
					TurtleID: "0",
				},
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
