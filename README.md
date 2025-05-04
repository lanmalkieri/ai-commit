# ai-commit

AI-powered Git commit message generator using OpenRouter API.

## Installation

### Building from Source

Clone the repository and build the binary:

```bash
# Clone the repository
git clone https://github.com/cstobie/ai-commit.git
cd ai-commit

# Build the binary
go build -o ai-commit .

# Install to your $GOPATH/bin
go install
```

### From GitHub Releases

Download the appropriate binary for your platform from the [GitHub Releases](https://github.com/cstobie/ai-commit/releases) page and place it in your PATH.

### Using Go Install

```bash
go install github.com/cstobie/ai-commit@latest
```

## Configuration

The tool is configured using environment variables, all prefixed with `AICOMMIT_`.

| Environment Variable          | Description                                           | Default Value      |
|-------------------------------|-------------------------------------------------------|--------------------|
| `AICOMMIT_OPENROUTER_API_KEY` | OpenRouter API key (required)                         | -                  |
| `AICOMMIT_LLM_MODEL`          | Model to use from OpenRouter                          | openai/gpt-4o-mini |
| `AICOMMIT_MAX_INPUT_TOKENS`   | Maximum tokens to send to the LLM                     | 4000               |
| `AICOMMIT_MAX_OUTPUT_TOKENS`  | Maximum tokens to generate for the commit message     | 200                |
| `AICOMMIT_TEMPLATE_NAME`      | Template name to use ("conventional" or "simple")     | conventional       |
| `AICOMMIT_TIMEOUT_SECONDS`    | Timeout for the API request in seconds               | 60                 |
| `AICOMMIT_TEMPERATURE`        | Temperature parameter for the LLM generation          | 0.7                |

## Usage

```bash
# Generate a commit message and prompt for commit confirmation (default behavior)
ai-commit

# Generate a message without interactive confirmation
ai-commit gen -n

# These commands are all equivalent (they generate a message and prompt for confirmation)
ai-commit
ai-commit generate 
ai-commit gen

# Show version information
ai-commit --version

# With verbose logging
ai-commit gen -v

# Use the simple template for this command
AICOMMIT_TEMPLATE_NAME=simple ai-commit gen
```

## Templates

The tool comes with two built-in templates:

1. **conventional** (default): Follows the [Conventional Commits](https://www.conventionalcommits.org/) specification
2. **simple**: Generates a short, plain text commit message

## Examples

```bash
# Stage some changes
git add src/feature.js

# Generate a commit message
ai-commit gen

# Output example:
# feat: add user authentication function with JWT support
```

## Development

### Running Locally

To run the tool during development without installing:

```bash
# Set required environment variables
export AICOMMIT_OPENROUTER_API_KEY=your_api_key_here

# Run the tool directly
go run main.go generate

# Run with verbose output
go run main.go generate -v
```

### Building for Distribution

This project uses GoReleaser for creating distribution packages:

```bash
# Install GoReleaser (if needed)
go install github.com/goreleaser/goreleaser@latest

# Build a snapshot release for testing
goreleaser build --snapshot --clean

# Full release (for maintainers only)
goreleaser release --clean
```

## Exit Codes

| Code | Description                                                |
|------|------------------------------------------------------------|  
| 0    | Success                                                    |
| 1    | General Error                                              |
| 2    | Git Error (not a repo, git command failed)                 |
| 3    | Configuration Error (invalid settings, missing API key)    |
| 4    | API Error (authentication failure, rate limit, timeout)    |
| 5    | Template Error (template not found, parsing error)         |

## License

MIT
