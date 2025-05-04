package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type OpenRouterMessage struct {
	Role    string `json:"role"` // "system" or "user"
	Content string `json:"content"`
}

type OpenRouterChatRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenRouterMessage `json:"messages"`
	Temperature *float64            `json:"temperature,omitempty"` // Pointer to allow omission
	MaxTokens   *int                `json:"max_tokens,omitempty"`  // Pointer for completion tokens
}

type OpenRouterChoice struct {
	Message OpenRouterMessage `json:"message"`
}

type OpenRouterChatResponse struct {
	ID      string             `json:"id"`
	Choices []OpenRouterChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"` // Can be string or int
	} `json:"error,omitempty"`
}

// EstimateTokens provides a simple word-based token estimation
func EstimateTokens(text string) int {
	return len(strings.Fields(text))
}

// TruncateInput truncates the prompt to fit within maxTokens
func TruncateInput(prompt string, maxTokens int) (string, bool) {
	tokens := EstimateTokens(prompt)
	if tokens <= maxTokens {
		return prompt, false
	}

	words := strings.Fields(prompt)
	keepTokens := maxTokens / 2
	
	// Keep the first and last parts of the prompt
	if len(words) > maxTokens {
		truncated := append(
			words[:keepTokens],
			append(
				[]string{"[...truncated...]"},
				words[len(words)-keepTokens:]...,
			)...,
		)
		return strings.Join(truncated, " "), true
	}
	
	return prompt, false
}

// GenerateCommitMessage calls the OpenRouter API to generate a commit message
func GenerateCommitMessage(ctx context.Context, apiKey, model string, maxOutputTokens int, 
	temperature float64, fullPrompt string, maxInputTokens int) (string, error) {
	
	// Truncate input if needed
	truncatedPrompt, wasTruncated := TruncateInput(fullPrompt, maxInputTokens)
	if wasTruncated {
		log.Println("Warning: Prompt was truncated to fit within token limits")
	}

	// Build request
	messages := []OpenRouterMessage{
		{Role: "user", Content: truncatedPrompt},
	}

	requestBody := OpenRouterChatRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   &maxOutputTokens,
		Temperature: &temperature,
	}

	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://openrouter.ai/api/v1/chat/completions",
		bytes.NewBuffer(requestBodyBytes),
	)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "github.com/cstobie/ai-commit")
	req.Header.Set("X-Title", "AI-Commit CLI")

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("request timed out: %w", ctx.Err())
		}
		return "", fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody := new(bytes.Buffer)
		_, _ = responseBody.ReadFrom(resp.Body)
		
		switch resp.StatusCode {
		case 401:
			return "", fmt.Errorf("API authentication error (code %d): %s", resp.StatusCode, responseBody.String())
		case 429:
			return "", fmt.Errorf("API rate limit exceeded (code %d): %s", resp.StatusCode, responseBody.String())
		default:
			if resp.StatusCode >= 500 {
				return "", fmt.Errorf("API server error (code %d): %s", resp.StatusCode, responseBody.String())
			}
			return "", fmt.Errorf("API error (code %d): %s", resp.StatusCode, responseBody.String())
		}
	}

	// Parse response
	var response OpenRouterChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	// Check for API errors in response body
	if response.Error != nil && response.Error.Message != "" {
		return "", fmt.Errorf("API error: %s", response.Error.Message)
	}

	// Extract and validate response content
	if len(response.Choices) == 0 || response.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("LLM returned empty response")
	}

	// Return the generated commit message
	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}