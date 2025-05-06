package git

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

// FileChange represents a single file change in git
type FileChange struct {
	Path      string // Full path to the file
	ChangeType string // Added, Modified, Deleted, Renamed
	IsBinary   bool   // Whether the file is binary
	Diff       string // The diff content for this file
}

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

// GetStagedDiffFiles parses git diff and returns structured file changes
func GetStagedDiffFiles(repoRoot string) ([]FileChange, error) {
	// Get raw diff
	diffOutput, err := GetStagedDiff(repoRoot)
	if err != nil {
		return nil, err
	}
	if diffOutput == "" {
		return []FileChange{}, nil
	}

	// Get list of changed files
	fileListCmd := exec.Command("git", "-C", repoRoot, "diff", "--staged", "--name-status")
	fileListOutput, err := fileListCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error getting staged file list: %w", err)
	}
	
	// Parse file list
	fileChanges := make([]FileChange, 0)
	fileListLines := strings.Split(strings.TrimSpace(string(fileListOutput)), "\n")
	
	// Regex to match diff headers
	diffHeaderRegex := regexp.MustCompile(`(?m)^diff --git a/(.+) b/(.+)$`)
	binaryFileRegex := regexp.MustCompile(`(?m)^Binary files`)
	
	for _, line := range fileListLines {
		if line == "" {
			continue
		}
		
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		
		changeType := parts[0]
		filePath := parts[1]
		
		// Handle rename case
		if changeType[0] == 'R' {
			renameParts := strings.SplitN(filePath, "\t", 2)
			if len(renameParts) == 2 {
				filePath = renameParts[1] // Use the new path
			}
		}
		
		// Map git status to change type
		var changeTypeStr string
		switch changeType[0] {
		case 'A':
			changeTypeStr = "Added"
		case 'M':
			changeTypeStr = "Modified"
		case 'D':
			changeTypeStr = "Deleted"
		case 'R':
			changeTypeStr = "Renamed"
		default:
			changeTypeStr = "Modified" // Default case
		}
		
		// Corresponding file blocks in the full diff
		fileChange := FileChange{
			Path:       filePath,
			ChangeType: changeTypeStr,
			IsBinary:   false, // Will be set below if binary
			Diff:       "",    // Will be set below
		}
		
		// Find this file's diff in the full diff output
		matches := diffHeaderRegex.FindAllStringSubmatchIndex(diffOutput, -1)
		for i, match := range matches {
			// Extract file path from the diff header
			startB := match[4]
			endB := match[5]
			
			filePathInDiff := diffOutput[startB:endB]
			
			// If this is our file
			if filePathInDiff == filePath || strings.HasSuffix(filePathInDiff, "/"+filePath) {
				// Find the start of this file's diff
				diffStart := match[0]
				
				// Find the end (next file or end of diff)
				diffEnd := len(diffOutput)
				if i < len(matches)-1 {
					diffEnd = matches[i+1][0]
				}
				
				// Extract this file's diff
				fileDiff := diffOutput[diffStart:diffEnd]
				
				// Check if binary
				if binaryFileRegex.MatchString(fileDiff) {
					fileChange.IsBinary = true
				}
				
				fileChange.Diff = fileDiff
				break
			}
		}
		
		fileChanges = append(fileChanges, fileChange)
	}
	
	return fileChanges, nil
}

// GetStagedFilesList returns a list of staged files with their status
func GetStagedFilesList(repoRoot string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "diff", "--staged", "--name-status")
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return "", fmt.Errorf("error getting staged files list: %w", err)
	}
	
	// An empty output is valid - it means no staged changes
	return string(output), nil
}

// PrepareSmartDiff creates an intelligent diff summary for large commits
// It ensures all files are included, with truncation applied based on file importance
func PrepareSmartDiff(repoRoot string, maxTokens int) (string, error) {
	// Get all file changes
	fileChanges, err := GetStagedDiffFiles(repoRoot)
	if err != nil {
		return "", err
	}
	
	if len(fileChanges) == 0 {
		return "", nil
	}
	
	// Create a summary of all files changed
	var sb strings.Builder
	
	// First, add a header with file statistics
	sb.WriteString(fmt.Sprintf("Commit includes %d files:\n", len(fileChanges)))
	
	// Count files by type
	added := 0
	modified := 0
	deleted := 0
	renamed := 0
	binary := 0
	
	for _, fc := range fileChanges {
		switch fc.ChangeType {
		case "Added":
			added++
		case "Modified":
			modified++
		case "Deleted":
			deleted++
		case "Renamed":
			renamed++
		}
		if fc.IsBinary {
			binary++
		}
	}
	
	// Add statistics
	sb.WriteString(fmt.Sprintf("- Added: %d\n", added))
	sb.WriteString(fmt.Sprintf("- Modified: %d\n", modified))
	sb.WriteString(fmt.Sprintf("- Deleted: %d\n", deleted))
	if renamed > 0 {
		sb.WriteString(fmt.Sprintf("- Renamed: %d\n", renamed))
	}
	if binary > 0 {
		sb.WriteString(fmt.Sprintf("- Binary files: %d\n", binary))
	}
	
	// Group files by directory to identify patterns
	dirGroups := make(map[string][]string)
	for _, fc := range fileChanges {
		// Extract directory from path
		pathParts := strings.Split(fc.Path, "/")
		var dir string
		if len(pathParts) > 1 {
			// For multi-level paths, use the first folder
			dir = pathParts[0]
		} else {
			// For files in root, use "root"
			dir = "root"
		}
		
		// Add to directory group
		dirGroups[dir] = append(dirGroups[dir], fc.Path)
	}
	
	// Add directory grouping information
	if len(dirGroups) > 1 {
		sb.WriteString("\nChanges by directory:\n")
		for dir, files := range dirGroups {
			sb.WriteString(fmt.Sprintf("- %s: %d files\n", dir, len(files)))
		}
	}
	
	// Add list of all files with change type
	sb.WriteString("\nChanged files:\n")
	for _, fc := range fileChanges {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", fc.ChangeType, fc.Path))
	}
	
	// Budget tokens per file
	// Reserve ~20% of tokens for the summary and metadata
	fileDiffBudget := int(float64(maxTokens) * 0.8)
	tokensPerFile := fileDiffBudget / len(fileChanges)
	
	// Log token budget info
	log.Printf("Smart diff processing: Total tokens=%d, Files=%d, Tokens per file=%d", 
		maxTokens, len(fileChanges), tokensPerFile)
	
	// Add selected diff content for each file
	sb.WriteString("\nSelected diff content:\n")
	
	// Process each file's diff
	for i, fc := range fileChanges {
		// Skip binary files
		if fc.IsBinary {
			sb.WriteString(fmt.Sprintf("\n### %s: %s (binary file)\n", fc.ChangeType, fc.Path))
			continue
		}
		
		// For text files, add a header
		sb.WriteString(fmt.Sprintf("\n### %s: %s\n", fc.ChangeType, fc.Path))
		
		// For deleted files, just note that they were deleted
		if fc.ChangeType == "Deleted" {
			sb.WriteString("File was deleted.\n")
			continue
		}
		
		// For other files, include a portion of the diff
		if fc.Diff != "" {
			// Simple tokenization (we'll estimate 1 token ≈ 4 characters)
			diffChars := len(fc.Diff)
			diffTokenEst := diffChars / 4
			
			if i < 5 {
				// Log details for first few files
				log.Printf("File %d: %s - Est. tokens: %d (budget: %d)", 
					i+1, fc.Path, diffTokenEst, tokensPerFile)
			}
			
			if diffChars > tokensPerFile*4 {
				// Extract a summary portion (start of the diff)
				diffLines := strings.Split(fc.Diff, "\n")
				if len(diffLines) > 5 {
					// Include first 5 lines
					sb.WriteString(strings.Join(diffLines[:5], "\n"))
					sb.WriteString("\n... (diff truncated) ...\n")
					
					// Also include snippets of functions or significant changes if present
					// Look for function definitions or significant patterns
					funcPattern := regexp.MustCompile(`(?m)^[+-](func|def|class|void|export|function)`)
					importPattern := regexp.MustCompile(`(?m)^[+-](import|from|require|use|using)`)
					
					// Find and include important chunks
					var importantChunks []string
					chunkStart := -1
					
					for i, line := range diffLines {
						if funcPattern.MatchString(line) || importPattern.MatchString(line) {
							// Found important line
							if chunkStart == -1 {
								chunkStart = i
							}
						} else if chunkStart != -1 && i > chunkStart+4 {
							// End of chunk (at least 5 lines after start)
							if chunkStart < i-1 {
								// Only include substantial chunks
								chunk := strings.Join(diffLines[chunkStart:i], "\n")
								importantChunks = append(importantChunks, chunk)
							}
							chunkStart = -1
						}
					}
					
					// Include up to 3 important chunks
					maxChunks := 3
					if len(importantChunks) > 0 {
						sb.WriteString("\nImportant changes:\n")
						for i, chunk := range importantChunks {
							if i >= maxChunks {
								break
							}
							sb.WriteString(chunk)
							sb.WriteString("\n---\n")
						}
					}
				} else {
					// Small diff, include it all
					sb.WriteString(fc.Diff)
				}
			} else {
				// Entire diff fits in budget
				sb.WriteString(fc.Diff)
			}
		} else {
			sb.WriteString("(No diff content available)\n")
		}
	}
	
	// Log summary
	finalOutput := sb.String()
	log.Printf("Smart diff processing complete: Generated summary of %d characters", len(finalOutput))
	
	return finalOutput, nil
}