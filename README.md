# Presto

Presto is a command-line tool that uses AI to process files and directories. Give it files and a prompt, and AI will transform or generate content according to your instructions.

## What is Presto?

Presto sends your files to AI models (like Claude or GPT-4) along with instructions on what you want done. The AI reads your files, follows your prompt, and returns the processed content. You can transform existing files or generate new ones.

**Transform mode**: Modify existing files

- "Add comments to this code"
- "Fix grammar in these documents"
- "Reformat this data"

**Generate mode**: Create new files from existing ones

- "Write documentation for this codebase"
- "Create a summary of these reports"
- "Generate tests based on this code"

## Why Use Presto?

- **Batch processing**: Transform hundreds of files at once
- **Consistency**: AI applies the same logic across all files
- **Context awareness**: AI can see multiple files to understand patterns
- **Automation**: Script repetitive file processing tasks
- **Flexibility**: Works with any text files - code, docs, data, etc.

## Installation

```bash
git clone https://github.com/yourusername/presto.git
cd presto
make build
sudo make install
```

Set your OpenRouter API key:

```bash
export OPENROUTER_API_KEY="your-key-here"
```

## Basic Usage

### Transform Files

Process a single file:

```bash
presto --prompt "Add detailed comments explaining what this code does" --input main.go
```

Process multiple files:

```bash
presto --prompt "Fix grammar and spelling errors" --input docs/ --recursive
```

The AI will read each file, apply your prompt, and save the results with a `.presto` suffix.

### Generate New Files

Create new content from existing files:

```bash
presto --generate \
    --prompt "Write comprehensive documentation for this project" \
    --context README.md \
    --context src/ \
    --output-file DOCUMENTATION.md
```

The AI reads the context files and generates new content based on your prompt.

## Command-Line Options

### Basic Options

- `--prompt TEXT` - Instructions for the AI
- `--input PATH` - File or directory to process
- `--recursive` - Process subdirectories
- `--dry-run` - Show what would be done without making changes
- `--verbose` - Show detailed progress

### File Filtering

Control which files get processed:

- `--pattern REGEX` - Only process files matching this pattern
- `--exclude REGEX` - Skip files matching this pattern

Examples:

```bash
# Only Python files
presto --prompt "Add type hints" --pattern "\.py$" --recursive

# Skip test files
presto --prompt "Add logging" --exclude "_test\.|_spec\." --recursive

# Specific file types
presto --prompt "Format consistently" --pattern "\.(js|ts|jsx)$" --input src/
```

### Output Control

Choose where results go:

- `--output separate` - Create new files with suffix (default)
- `--output inplace` - Modify original files
- `--output stdout` - Print results to terminal
- `--output file` - Single output file (generate mode only)

Additional output options:

- `--suffix TEXT` - Custom suffix for separate mode (default: `.presto`)
- `--output-file PATH` - Specific output file for generate mode
- `--backup` - Create backups when using inplace mode

Examples:

```bash
# Create new files with .fixed suffix
presto --prompt "Fix bugs" --output separate --suffix ".fixed" --input src/

# Modify files in place with backup
presto --prompt "Reformat code" --output inplace --backup --recursive

# Print results to terminal
presto --prompt "Extract function names" --output stdout --input utils.py
```

### AI Context

Provide additional files to help AI understand your project:

- `--context FILE` - Single context file
- `--context-pattern PATTERN` - Multiple files matching pattern

The AI sees both the files being processed AND the context files, giving better results.

Examples:

```bash
# Use style guide as context
presto --prompt "Match the coding style in the guide" \
    --context docs/style-guide.md \
    --input src/ --recursive

# Use existing good examples
presto --prompt "Follow the same patterns as the examples" \
    --context-pattern "examples/*.py" \
    --input new-feature/

# Multiple context sources
presto --prompt "Maintain consistency with existing code" \
    --context config.py \
    --context-pattern "models/*.py" \
    --input new-models/
```

### AI Model Options

- `--model NAME` - Specific AI model to use
- `--temperature NUMBER` - Creativity level (0.0-2.0, lower = more focused)
- `--max-tokens NUMBER` - Maximum response length

Examples:

```bash
# Use specific model
presto --prompt "Complex refactoring task" --model "anthropic/claude-3.5-sonnet"

# Low creativity for precise tasks
presto --prompt "Fix syntax errors" --temperature 0.0 --recursive

# High creativity for creative writing
presto --prompt "Make this text more engaging" --temperature 1.5 --input articles/
```

### Performance Options

- `--concurrent NUMBER` - How many files to process simultaneously (default: 3)

```bash
# Process many files faster
presto --prompt "Add headers" --concurrent 8 --recursive --input large-codebase/

# Single-threaded for debugging
presto --prompt "Complex task" --concurrent 1 --verbose --input src/
```

## Built-in Commands

Pre-made prompts and settings for common tasks:

### Available Commands

- `add-docs` - Add documentation and comments
- `add-logging` - Add logging statements
- `optimize` - Improve performance and readability
- `modernize` - Update to current best practices
- `cleanup` - Clean formatting and remove clutter
- `explain` - Add explanatory comments
- `summarize` - Create summaries (generates new files)
- `convert` - Convert to different format

### Using Built-in Commands

```bash
# Use a command instead of writing a prompt
presto --cmd add-docs --input src/ --recursive

# See all available commands
presto --list-commands

# See what a command does
presto --show-command add-docs
```

Commands are shortcuts that include both a prompt and useful default settings.

## Custom Commands

Save your own prompts as reusable commands.

### Why Create Custom Commands?

- Reuse complex prompts across projects
- Share common tasks with your team
- Include default settings and file patterns
- Use variables for flexibility

### Creating Custom Commands

```bash
# Save current settings as a command
presto --prompt "Add comprehensive error handling with logging and metrics" \
    --pattern "\.go$" \
    --exclude "_test\.go$" \
    --recursive \
    --save-command add-error-handling

# Use your saved command
presto --cmd add-error-handling --input .
```

### Commands with Variables

```bash
# Create command with placeholder variables
presto --prompt "Convert this {{SOURCE}} to {{TARGET}} format" \
    --var SOURCE=yaml \
    --var TARGET=json \
    --save-command convert-format

# Use with different values
presto --cmd convert-format \
    --var SOURCE=xml \
    --var TARGET=yaml \
    --input configs/
```

### Managing Custom Commands

```bash
# List all commands
presto --list-commands

# Show command details
presto --show-command add-error-handling

# Delete a command
presto --delete-command convert-format

# Edit a command (creates template file)
presto --edit-command my-command
```

Commands are saved in `~/.presto/commands/` as YAML files you can edit directly.

## Generate Mode

Create new files using existing files as context.

### When to Use Generate Mode

- Create documentation from code
- Generate summaries from multiple files
- Create new files following existing patterns
- Extract or transform information across files

### Generate Mode Examples

```bash
# Create project documentation
presto --generate \
    --prompt "Write comprehensive documentation for this Go project" \
    --context-pattern "*.go" \
    --context README.md \
    --output-file docs/API.md

# Generate summary report
presto --generate \
    --prompt "Create executive summary of these quarterly reports" \
    --context-pattern "reports/2024-Q*.txt" \
    --output-file summary.md

# Create new file following patterns
presto --generate \
    --prompt "Create a user service following the same patterns as existing services" \
    --context-pattern "services/*service.go" \
    --output-file services/user-service.go
```

Generate mode requires:

- `--generate` flag
- `--output-file PATH` for the new file
- Context files (via `--context` or `--context-pattern`)

## Practical Examples

### Documentation Tasks

```bash
# Add comments to uncommented code
presto --cmd add-docs --pattern "\.py$" --recursive

# Generate API documentation from code
presto --generate \
    --prompt "Create OpenAPI specification from these route handlers" \
    --context-pattern "routes/*.js" \
    --output-file api-spec.yaml

# Create README from existing files
presto --generate \
    --prompt "Write a comprehensive README for this project" \
    --context-pattern "*.py" \
    --context requirements.txt \
    --output-file README.md
```

### Code Improvement

```bash
# Clean up messy code
presto --cmd cleanup --input legacy/ --recursive

# Add error handling consistently
presto --prompt "Add proper error handling following Go best practices" \
    --context-pattern "*error*.go" \
    --pattern "\.go$" \
    --exclude "_test\.go$" \
    --recursive

# Modernize old code with examples
presto --prompt "Update this code to use modern Python practices" \
    --context examples/modern-style.py \
    --pattern "\.py$" \
    --input legacy/
```

### Content Processing

```bash
# Fix formatting across documents
presto --prompt "Fix formatting, grammar, and spelling" \
    --pattern "\.md$" \
    --recursive

# Convert file formats
presto --prompt "Convert this YAML to JSON format" \
    --pattern "\.ya?ml$" \
    --suffix ".json"

# Standardize data formats
presto --prompt "Convert all dates to ISO 8601 format" \
    --pattern "data.*\.txt$" \
    --recursive
```

### Analysis and Extraction

```bash
# Extract specific information
presto --prompt "Extract all TODO comments and create a task list" \
    --recursive \
    --output stdout > todo-list.txt

# Analyze patterns across files
presto --generate \
    --prompt "Analyze the error handling patterns and suggest improvements" \
    --context-pattern "*.go" \
    --output-file error-analysis.md

# Create summaries
presto --cmd summarize \
    --context-pattern "meeting-notes/*.txt" \
    --output-file weekly-summary.md
```

## Configuration

Create `~/.presto/config.yaml` to set default options:

```yaml
# AI settings
ai:
  model: "anthropic/claude-3.5-sonnet"
  max_tokens: 4000
  temperature: 0.1
  timeout: 60s

# Default behavior
defaults:
  max_concurrent: 3
  output_mode: "separate"
  output_suffix: ".presto"
  backup_original: true

# File filtering
filters:
  max_file_size: 1048576 # 1MB max file size
  exclude_dirs:
    - ".git"
    - "node_modules"
    - "vendor"
    - "__pycache__"
  exclude_exts:
    - ".exe"
    - ".bin"
    - ".so"
```

Configuration options override built-in defaults but are overridden by command-line flags.

## Tips and Best Practices

### Getting Better Results

1. **Be specific in prompts**: "Add detailed docstrings following PEP 257" vs "add docs"
2. **Provide context**: Include style guides, examples, or related files
3. **Use appropriate models**: Complex tasks need better models
4. **Start small**: Test on a few files before processing large directories

### File Processing Tips

1. **Use patterns**: Be selective about which files to process
2. **Check outputs**: Always review AI changes before committing
3. **Use dry-run**: Preview changes with `--dry-run --verbose`
4. **Make backups**: Use `--backup` with inplace mode

### Performance Tips

1. **Adjust concurrency**: More concurrent jobs = faster processing
2. **Filter aggressively**: Skip unnecessary files with patterns
3. **Use appropriate context**: Don't include irrelevant context files
4. **Choose efficient models**: Faster models for simple tasks

## Troubleshooting

### Common Issues

**"No API key found"**

```bash
# Check if key is set
echo $OPENROUTER_API_KEY

# Set the key
export OPENROUTER_API_KEY="your-key-here"
```

**"No files found to process"**

```bash
# Check what files match your pattern
presto --prompt "test" --pattern "\.py$" --dry-run --verbose

# Verify file paths
ls -la your-input-path/
```

**"API request failed"**

- Check your internet connection
- Verify your API key is valid
- Try a different model with `--model`

**Poor AI results**

- Make prompts more specific
- Add more context files
- Try a higher-quality model
- Lower temperature for more focused results

### Getting Help

```bash
# Show all options
presto --help

# List available commands
presto --list-commands

# Show what a command does
presto --show-command COMMAND_NAME
```

### Debug Mode

Use these flags to understand what's happening:

```bash
# See what files would be processed
presto --dry-run --verbose --recursive --input .

# Test with a single file first
presto --prompt "test" --input single-file.txt --verbose

# Check AI response without saving
presto --prompt "test" --input file.txt --output stdout
```
