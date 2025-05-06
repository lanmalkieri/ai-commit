package app

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/cstobie/ai-commit/internal/config"
	"github.com/cstobie/ai-commit/internal/git"
	"github.com/cstobie/ai-commit/internal/llm"
	"github.com/cstobie/ai-commit/internal/template"
)

// RunGenerate orchestrates the commit message generation process
func RunGenerate(ctx context.Context, cfg config.Config, verbose bool, interactive bool) error {
	// Step 1: Find the git repository root
	repoRoot, err := git.GetRepoRoot(".")
	if err != nil {
		return fmt.Errorf("This command must be run inside a git repository. %w", err)
	}
	
	if verbose {
		log.Printf("Found git repository at: %s", repoRoot)
	}

	// Step 2: Get the staged diff (check if using smart diff for large commits)
	var diff string
	// First, get a quick count of changed files
	filesList, err := git.GetStagedFilesList(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to get staged files list: %w", err)
	}
	
	// Count files by counting newlines
	fileCount := 0
	if filesList != "" {
		fileCount = len(strings.Split(strings.TrimSpace(filesList), "\n"))
	}
	
	// Check if there are any staged changes
	if filesList == "" {
		fmt.Println("No staged changes found. Stage changes first with 'git add'.")
		return nil
	}
	
	// For multi-file commits, use smart diff to preserve context
	if fileCount > 5 { // Threshold for "large" commits
		if verbose {
			log.Printf("Large commit detected (%d files). Using smart diff processing.", fileCount)
		}
		// Use the smart diff processor with the configured token limit
		smartDiff, err := git.PrepareSmartDiff(repoRoot, cfg.MaxInputTokens)
		if err != nil {
			return fmt.Errorf("failed to prepare smart diff: %w", err)
		}
		diff = smartDiff
	} else {
		// For smaller commits, use the standard diff
		standardDiff, err := git.GetStagedDiff(repoRoot)
		if err != nil {
			return fmt.Errorf("failed to get staged changes: %w", err)
		}
		diff = standardDiff
	}
	
	if verbose {
		log.Printf("Retrieved staged diff (%d characters)", len(diff))
	}

	// Step 3: Load and execute the template
	fullPrompt, err := template.LoadAndExecuteTemplate(cfg.TemplateName, diff)
	if err != nil {
		return fmt.Errorf("failed to prepare prompt: %w", err)
	}
	
	if verbose {
		log.Printf("Using template: %s", cfg.TemplateName)
		log.Printf("Prepared prompt (%d characters)", len(fullPrompt))
	}

	// Step 4: Generate commit message using the LLM
	generatedMsg, err := llm.GenerateCommitMessage(
		ctx,
		cfg.OpenRouterAPIKey,
		cfg.LLMModel,
		cfg.MaxOutputTokens,
		cfg.Temperature,
		fullPrompt,
		cfg.MaxInputTokens,
	)
	
	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}

	// Step 5: Print the generated message
	fmt.Println("Generated commit message:")
	fmt.Println("---")
	fmt.Println(generatedMsg)
	fmt.Println("---")
	
	// Step 6: Handle interactive flow or not
	if interactive {
		// Verify that there are changes to commit
		if diff == "" {
			fmt.Println("No staged changes to commit. Stage changes first with 'git add'.")
			return nil
		}
		
		// Prompt for confirmation
		fmt.Print("Press Enter to commit with this message (or any key to abort): ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)
		
		// Process response - empty means Enter was pressed
		if response == "" {
			// User confirmed, proceed with commit
			if err := performCommit(repoRoot, generatedMsg, verbose); err != nil {
				return err
			}
		} else {
			fmt.Println("Commit aborted.")
		}
	} else {
		// Just print the message in non-interactive mode
		if verbose {
			log.Println("Running in non-interactive mode, message generated but not committed.")
		}
	}
	
	return nil
}

// performCommit executes the git commit with the provided message
func performCommit(repoRoot, message string, verbose bool) error {
	if verbose {
		log.Println("Committing changes with the generated message...")
	}
	
	// Create a temporary file to store the commit message
	tmpFile, err := os.CreateTemp("", "ai-commit-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for commit message: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	
	// Write the commit message to the temporary file
	if _, err := tmpFile.WriteString(message); err != nil {
		return fmt.Errorf("failed to write commit message to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	
	// Execute the git commit command using the file
	cmd := exec.Command("git", "-C", repoRoot, "commit", "-F", tmpFile.Name())
	commitOutput, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w\n%s", err, string(commitOutput))
	}
	
	if verbose {
		log.Printf("Commit successful:\n%s", string(commitOutput))
	} else {
		fmt.Println("Changes committed successfully!")
	}
	
	return nil
}