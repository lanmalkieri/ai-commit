package cmd

import (
	"fmt"
	"log"

	"github.com/cstobie/ai-commit/internal/config"
	"github.com/spf13/cobra"
)

// Version information
const version = "0.1.0"

// Global configuration variable
var cfg config.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ai-commit",
	Short: "AI-powered Git commit message generator",
	Long: `ai-commit is a CLI tool that generates commit messages using OpenRouter API.

It analyzes staged Git changes and suggests a well-formatted commit message
based on the selected template style.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Add the generate command
	rootCmd.AddCommand(generateCmd)
	
	// Add version flag
	rootCmd.Flags().BoolP("version", "V", false, "Print version information and exit")
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		versionFlag, _ := cmd.Flags().GetBool("version")
		if versionFlag {
			fmt.Printf("ai-commit version %s\n", version)
			return nil
		}
		
		// If no version flag or other command, run the generate command by default
		// This makes `ai-commit` behave the same as `ai-commit generate`
		return generateCmd.RunE(generateCmd, args)
	}
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	var err error
	cfg, err = config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
}
