package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	OpenRouterAPIKey string  `mapstructure:"OPENROUTER_API_KEY"`
	LLMModel         string  `mapstructure:"LLM_MODEL"`
	MaxInputTokens   int     `mapstructure:"MAX_INPUT_TOKENS"`
	MaxOutputTokens  int     `mapstructure:"MAX_OUTPUT_TOKENS"`
	TemplateName     string  `mapstructure:"TEMPLATE_NAME"`
	BasePrompt       string  `mapstructure:"BASE_PROMPT"` // Internal use for template
	TimeoutSeconds   int     `mapstructure:"TIMEOUT_SECONDS"`
	Temperature      float64 `mapstructure:"TEMPERATURE"` // Optional temperature setting
}

func LoadConfig() (Config, error) {
	viper.SetEnvPrefix("AICOMMIT") // Environment variables prefix: AICOMMIT_
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicitly bind each config key to its environment variable
	viper.BindEnv("OPENROUTER_API_KEY")
	viper.BindEnv("LLM_MODEL")
	viper.BindEnv("MAX_INPUT_TOKENS")
	viper.BindEnv("MAX_OUTPUT_TOKENS")
	viper.BindEnv("TEMPLATE_NAME")
	viper.BindEnv("TIMEOUT_SECONDS")
	viper.BindEnv("TEMPERATURE")

	// Default values
	viper.SetDefault("LLM_MODEL", "openai/gpt-4o-mini") // Updated Default Model
	viper.SetDefault("MAX_INPUT_TOKENS", 4000)
	viper.SetDefault("MAX_OUTPUT_TOKENS", 200)
	viper.SetDefault("TEMPLATE_NAME", "conventional")
	viper.SetDefault("TIMEOUT_SECONDS", 60) // Default request timeout
	viper.SetDefault("TEMPERATURE", 0.7)    // Default temperature

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unable to decode config: %w", err)
	}

	// Check if API key is loaded from environment
	_ = viper.GetString("OPENROUTER_API_KEY")

	// Validation (Example)
	if cfg.OpenRouterAPIKey == "" {
		log.Println("Warning: AICOMMIT_OPENROUTER_API_KEY environment variable not set.")
		// Allow proceeding but API calls will fail later if key is truly needed
	}
	if cfg.MaxInputTokens <= 0 || cfg.MaxOutputTokens <= 0 {
		return Config{}, fmt.Errorf("token limits must be positive")
	}

	return cfg, nil
}