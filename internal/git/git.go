package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetRepoRoot finds the root directory of the git repository containing the specified directory
func GetRepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	output, err := cmd.CombinedOutput()

	if err != nil {
		if _, err := exec.LookPath("git"); err != nil {
			return "", fmt.Errorf("git command not found: %w", err)
		}
		return "", fmt.Errorf("not a git repository or git error: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetStagedDiff returns the diff of all staged changes in the repository
func GetStagedDiff(repoRoot string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "diff", "--staged", "--patch", "--unified=0", 
		"--no-color", "--no-ext-diff", "--ignore-space-change", "--ignore-all-space", "--ignore-blank-lines")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("error getting staged diff: %w", err)
	}

	// An empty output is valid - it means no staged changes
	return string(output), nil
}