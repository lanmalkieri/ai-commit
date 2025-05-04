package cmd

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	"github.com/cstobie/ai-commit/internal/app"
	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "Generate commit message for staged changes",
	Long: `Generate commit message for staged changes based on the specified template.

Examples:
  ai-commit generate
  ai-commit gen -v
  AICOMMIT_TEMPLATE_NAME=simple ai-commit gen`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flag values
		verbose, _ := cmd.Flags().GetBool("verbose")
		noInteractive, _ := cmd.Flags().GetBool("no-interactive")
		
		// Configure logging based on verbose flag
		if !verbose {
			log.SetOutput(io.Discard)
		}

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(
			context.Background(), 
			time.Duration(cfg.TimeoutSeconds)*time.Second,
		)
		defer cancel()

		// Run the generate command with interactive mode by default
		interactive := !noInteractive
		return app.RunGenerate(ctx, cfg, verbose, interactive)
	},
}

func init() {
	// Define flags
	generateCmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging")
	generateCmd.Flags().BoolP("no-interactive", "n", false, "Generate message without interactive confirmation")
}
