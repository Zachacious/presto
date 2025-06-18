# Presto

> ⚠️ **Important**: Presto is designed for **plaintext file formats only** (code, configuration, documentation, etc.). It does not support binary formats like `.xlsx`, `.docx`, `.pdf`, images, or other non-text files. Always ensure your input files are text-based for proper processing.

**AI-Powered File Processor** - Transform, generate, and enhance your codebase using AI

Presto streamlines your development workflow by applying AI transformations to files and directories. Whether you're adding documentation, modernizing code, or generating new content, Presto makes it simple and consistent.

## 🎯 What Presto Does

- **Transform existing files** with AI-powered modifications
- **Generate new content** from context and prompts
- **Batch process** entire directories with intelligent filtering
- **Preserve your workflow** with flexible output modes and automatic backups
- **Use predefined commands** for common tasks or create custom ones

## 🚀 Quick Start

### Installation

```bash
# Install from source
git clone https://github.com/Zachacious/presto.git
cd presto
go build -o presto ./cmd/presto
sudo mv presto /usr/local/bin/
```

### First Time Setup

```bash
# Interactive configuration - sets up API keys and preferences
presto configure
```

Or set environment variables:

```bash
export OPENAI_API_KEY="your-api-key-here"
# or
export ANTHROPIC_API_KEY="your-anthropic-key"
```

### Basic Usage

```bash
# Default: Safe in-place editing with automatic backup
presto --prompt "Add comprehensive documentation" --input main.go

# Use a predefined command on entire project
presto --cmd add-docs --input . --recursive

# Generate a README from your codebase
presto --generate --prompt "Create comprehensive README" --context "*.go,*.md" --output-file README.md
```

## 🎪 Real-World Examples

### 1. Legacy Code Modernization

Transform an old JavaScript project to modern ES6+:

```bash
# Modernize all JavaScript files with smart suffix (preserves .js extension)
presto --cmd modernize \
  --input ./src \
  --recursive \
  --pattern ".*\.(js|jsx)$" \
  --output separate \
  --smart-suffix \
  --suffix .modern

# Result: Creates main.modern.js, utils.modern.js, etc.
```

**Before (utils.js):**

```javascript
var utils = {
  forEach: function (arr, callback) {
    for (var i = 0; i < arr.length; i++) {
      callback(arr[i], i);
    }
  },
  map: function (arr, callback) {
    var result = [];
    for (var i = 0; i < arr.length; i++) {
      result.push(callback(arr[i], i));
    }
    return result;
  },
};
```

**After (utils.modern.js):**

```javascript
const utils = {
  forEach: (arr, callback) => {
    arr.forEach((item, index) => callback(item, index));
  },

  map: (arr, callback) => {
    return arr.map((item, index) => callback(item, index));
  },
};

export default utils;
```

### 2. Adding Comprehensive Documentation

Document an entire Go microservice with automatic backup:

```bash
# Add docs to all Go files (default: backup + in-place modification)
presto --cmd add-docs \
  --input ./internal \
  --recursive \
  --context README.md,go.mod
```

**Before (user_service.go):**

```go
type UserService struct {
    db Database
    cache Cache
}

func (s *UserService) GetUser(id string) (*User, error) {
    if user := s.cache.Get(id); user != nil {
        return user, nil
    }
    user, err := s.db.FindUser(id)
    if err != nil {
        return nil, err
    }
    s.cache.Set(id, user)
    return user, nil
}
```

**After (creates user_service.go.backup, modifies original):**

```go
// UserService handles user-related operations with caching support.
// It implements a cache-aside pattern for optimal performance.
type UserService struct {
    db    Database // Primary database connection
    cache Cache    // In-memory cache for frequently accessed users
}

// GetUser retrieves a user by ID, checking cache first for performance.
//
// The method implements a cache-aside pattern:
// 1. Check cache for existing user data
// 2. If not found, query database
// 3. Cache the result for future requests
//
// Parameters:
//   - id: Unique user identifier (must be non-empty)
//
// Returns:
//   - *User: User object if found
//   - error: Database error or validation error if ID is invalid
func (s *UserService) GetUser(id string) (*User, error) {
    // Fast path: check cache first
    if user := s.cache.Get(id); user != nil {
        return user, nil
    }

    // Slow path: query database
    user, err := s.db.FindUser(id)
    if err != nil {
        return nil, fmt.Errorf("failed to find user %s: %w", id, err)
    }

    // Cache the result for future requests
    s.cache.Set(id, user)
    return user, nil
}
```

### 3. Parallel Directory Processing

Create enhanced versions while preserving originals:

```bash
# Create parallel enhanced directory structure
presto --prompt "Add structured logging for debugging and monitoring" \
  --input ./api \
  --recursive \
  --output directory \
  --output-dir ./enhanced-api

# Result: Creates ./enhanced-api/ with same structure as ./api/
```

### 4. Preview Mode for Safe Exploration

See changes before committing:

```bash
# Preview changes and choose where to save them
presto --prompt "Optimize for performance and add error handling" \
  --input complex_service.py \
  --preview

# Interactive options:
# 1. Save in-place (replace original)
# 2. Save with backup (.backup)
# 3. Save as separate file (.presto)
# 4. Save to custom file
# 5. Skip this file
```

### 5. Generating Project Documentation

Create comprehensive project documentation from your codebase:

```bash
# Generate README from all source files
presto --generate \
  --prompt "Create a comprehensive README.md with setup instructions, API documentation, and examples" \
  --context "*.go,*.sql,docker-compose.yml" \
  --output-file README.md

# Generate API documentation
presto --generate \
  --prompt "Create OpenAPI/Swagger documentation for this REST API" \
  --context "handlers/*.go,models/*.go" \
  --output-file api-docs.yaml
```

### 6. Batch Testing Generation

Generate comprehensive tests with smart naming:

```bash
# Generate unit tests for all services (smart suffix preserves .go extension)
presto --prompt "Generate comprehensive unit tests with mocking, edge cases, and good coverage" \
  --input ./services \
  --pattern ".*\.go$" \
  --exclude ".*_test\.go$" \
  --output separate \
  --suffix _test \
  --smart-suffix

# Result: user_service.go → user_service_test.go
```

## 🎛️ Output Modes

Presto offers flexible output modes to fit different workflows:

### In-Place (Default - Safe)

```bash
# Default: Creates backup, modifies original
presto --prompt "Add comments" --input main.go
# Creates: main.go.backup (original), main.go (modified)

# Explicit in-place without backup (use with caution)
presto --prompt "Add comments" --input main.go --output inplace
```

### Directory Mode (Parallel Structure)

```bash
# Create enhanced version in parallel directory
presto --cmd modernize \
  --input ./src \
  --output directory \
  --output-dir ./modernized

# Preserves: ./src/utils/helper.js
# Creates: ./modernized/src/utils/helper.js
```

### Separate Files (Smart Suffix)

```bash
# Smart suffix (before extension) - RECOMMENDED
presto --cmd add-docs \
  --input . \
  --output separate \
  --smart-suffix \
  --suffix .documented

# main.go → main.documented.go (preserves .go extension!)

# Traditional suffix (after extension)
presto --cmd add-docs \
  --input . \
  --output separate \
  --suffix .presto

# main.go → main.go.presto
```

### Single File Output

```bash
# For generate mode
presto --generate \
  --prompt "Create project summary" \
  --context "*.md,*.go" \
  --output-file SUMMARY.md
```

### Stdout (Pipe-Friendly)

```bash
# Print to terminal
presto --prompt "minify this" --input script.js --output stdout

# Pipe to other tools
presto --prompt "extract function names" --input app.py --output stdout | grep "def "
```

### Preview Mode (Interactive)

```bash
# See changes first, then decide where to save
presto --cmd optimize --input complex_algorithm.py --preview
```

## 🎭 Built-in Commands

### Code Enhancement Commands

```bash
# Add comprehensive documentation
presto --cmd add-docs --input ./src --recursive

# Add structured logging throughout codebase
presto --cmd add-logging --input ./api --recursive

# Optimize for performance and readability
presto --cmd optimize --input ./utils --recursive

# Modernize to current best practices
presto --cmd modernize --input ./legacy --recursive

# Clean up formatting and remove dead code
presto --cmd cleanup --input . --recursive
```

### Analysis and Documentation Commands

```bash
# Add explanatory comments and clarifications
presto --cmd explain --input complex_algorithm.py

# Generate project summary
presto --cmd summarize --context "*.md,*.go" --output-file SUMMARY.md

# Convert between formats/languages
presto --cmd convert --var TARGET_FORMAT=typescript --input api.js
```

## ⚙️ Configuration

### Config File Location

- **Linux/Mac:** `~/.presto/config.yaml`
- **Windows:** `%USERPROFILE%\.presto\config.yaml`

### Example Configuration

```yaml
ai:
  provider: openai
  api_key: "your-api-key"
  base_url: "https://api.openai.com/v1"
  model: "gpt-4"
  max_tokens: 4000
  temperature: 0.1
  timeout_seconds: 60

defaults:
  max_concurrent: 3
  output_mode: "inplace" # Default to safe in-place with backup
  output_suffix: ".presto"
  smart_suffix: true # Use smart suffix by default
  backup_original: true # Always backup by default
  remove_comments: false

filters:
  max_file_size: 1048576 # 1MB
  exclude_dirs:
    - ".git"
    - "node_modules"
    - "__pycache__"
    - "vendor"
  exclude_exts:
    - ".exe"
    - ".bin"
    - ".jpg"
    - ".png"
    - ".xlsx" # Excel files (not supported)
    - ".docx" # Word files (not supported)
    - ".pdf" # PDF files (not supported)
```

## 🎪 Creating Custom Commands

### Save Current Options as Command

```bash
# Create a custom TypeScript documentation command
presto --prompt "Add JSDoc comments with TypeScript types" \
  --pattern ".*\.(js|jsx)$" \
  --output separate \
  --smart-suffix \
  --suffix .documented \
  --save-command ts-docs

# Use your custom command
presto --cmd ts-docs --input ./src --recursive
```

### Command Templates

Commands are stored as YAML files in `~/.presto/commands/`:

```yaml
name: "add-api-docs"
description: "Add OpenAPI/Swagger documentation to REST endpoints"
mode: "transform"
prompt: |
  Add comprehensive OpenAPI/Swagger documentation to this {{LANGUAGE}} file.
  Include parameter descriptions, response schemas, and example requests/responses.
  Follow OpenAPI 3.0 specifications.
options:
  output_mode: "separate"
  smart_suffix: true
  output_suffix: ".documented"
  recursive: true
  context_patterns: ["*.yaml", "*.json"]
variables:
  LANGUAGE: "javascript"
```

## 🏗️ Advanced Workflows

### CI/CD Integration

```bash
#!/bin/bash
# .github/workflows/documentation.yml

# Auto-generate documentation on pull requests
presto --cmd add-docs \
  --input ./src \
  --recursive \
  --output directory \
  --output-dir ./docs-preview

# Check for outdated documentation
presto --prompt "Compare this code with its documentation and flag any inconsistencies" \
  --input ./src \
  --context "*.md" \
  --output-file doc-analysis.md
```

### Team Consistency

```bash
# Standardize code style across team (safe with backup)
presto --prompt "Ensure this code follows our team's style guide" \
  --input . \
  --recursive \
  --context ".eslintrc,.prettierrc,STYLE_GUIDE.md"
```

### Migration Assistance

```bash
# Migrate from Vue 2 to Vue 3 (parallel structure for comparison)
presto --prompt "Migrate this Vue 2 component to Vue 3 Composition API" \
  --input ./components \
  --pattern ".*\.vue$" \
  --context "package.json,migration-notes.md" \
  --output directory \
  --output-dir ./vue3-components
```

## 🎯 Supported AI Providers

### OpenAI

```bash
export OPENAI_API_KEY="your-key"
# Models: gpt-4, gpt-4-turbo, gpt-3.5-turbo
```

### Anthropic

```bash
export ANTHROPIC_API_KEY="your-key"
# Models: claude-3-5-sonnet, claude-3-haiku
```

### Local APIs (Ollama, LM Studio, etc.)

```bash
# No API key needed for local models
presto configure  # Choose "Local" option
# Set custom base URL: http://localhost:11434/v1
```

### Custom APIs

```bash
# Any OpenAI-compatible API
export PRESTO_BASE_URL="https://your-api.com/v1"
export PRESTO_API_KEY="your-key"
```

## 🎪 Tips and Best Practices

### 1. Start Small and Safe

```bash
# Test on a single file first with preview
presto --prompt "Add comments" --input example.js --preview
```

### 2. Use Smart Suffix for Better Tooling

```bash
# Preserves file extensions for better IDE/tooling support
presto --cmd add-docs \
  --input ./src \
  --output separate \
  --smart-suffix \
  --suffix .enhanced
```

### 3. Leverage Default Safety

```bash
# Default behavior is safe - creates backups automatically
presto --cmd modernize --input legacy-code.js
# Creates: legacy-code.js.backup, modifies legacy-code.js
```

### 4. Parallel Processing for Experimentation

```bash
# Create parallel enhanced version for comparison
presto --cmd optimize \
  --input ./current-api \
  --output directory \
  --output-dir ./optimized-api
```

### 5. Use Context Wisely

```bash
# Include relevant context for better results
presto --cmd add-docs \
  --input ./api \
  --context "README.md,package.json,api-spec.yaml"
```

### 6. Preview Mode for Critical Code

```bash
# Always preview changes to critical systems
presto --cmd modernize \
  --input ./payment-service \
  --preview
```

## 🚀 Performance Tips

- **Use file patterns** to avoid processing unnecessary files
- **Adjust concurrency** based on your system and API limits
- **Set reasonable token limits** to control costs
- **Use context selectively** - too much context can confuse the AI
- **Use smart suffix** for better tooling compatibility
- **Leverage parallel processing** for experimentation without risk

## 🎭 Troubleshooting

### Common Issues

**"API key not found"**

```bash
presto configure  # Run interactive setup
```

**"No files found"**

```bash
presto --input . --pattern ".*\.js$" --verbose  # Check file patterns
```

**"Binary file detected"**

```bash
# Presto only supports text files. Check your file pattern:
--pattern ".*\.(js|py|go|java|md|txt|yaml|json)$"
```

**"Request timeout"**

```bash
# Increase timeout in config or use smaller files
--max-tokens 2000
```

**"Rate limit exceeded"**

```bash
# Reduce concurrency
--concurrent 1
```

### Debug Mode

```bash
presto --verbose --dry-run  # See what will be processed
```

---

## Quick Reference

```bash
# ======================
# OUTPUT MODES
# ======================

# Default: Safe in-place with backup
presto --prompt "Add docs" --input main.go
# Creates: main.go.backup + modifies main.go

# Directory: Parallel structure
presto --cmd modernize --input ./src --output directory --output-dir ./enhanced

# Separate: Smart suffix (RECOMMENDED)
presto --cmd add-docs --input . --output separate --smart-suffix --suffix .enhanced
# Creates: main.enhanced.go (preserves .go extension)

# Separate: Traditional suffix
presto --cmd add-docs --input . --output separate --suffix .presto
# Creates: main.go.presto

# File: Single output (generate mode)
presto --generate --prompt "Create README" --context "*.go" --output-file README.md

# Stdout: Terminal output
presto --prompt "extract functions" --input app.py --output stdout

# Preview: Interactive mode
presto --cmd optimize --input critical.py --preview

# ======================
# BASIC USAGE
# ======================

# Interactive setup
presto configure

# Basic transformation (safe default)
presto --prompt "Your instruction" --input file.js

# Recursive processing
presto --prompt "Add comments" --input . --recursive

# Use predefined commands
presto --cmd COMMAND_NAME --input PATH [options]

# Generate new content
presto --generate --prompt "Create README" --context "*.go" --output-file README.md

# ======================
# BUILT-IN COMMANDS
# ======================

presto --cmd add-docs      # Add comprehensive documentation
presto --cmd add-logging   # Add structured logging statements
presto --cmd optimize      # Optimize for performance/readability
presto --cmd modernize     # Update to current best practices
presto --cmd cleanup       # Clean and format code
presto --cmd explain       # Add explanatory comments
presto --cmd summarize     # Create summary (generate mode)
presto --cmd convert       # Convert format/language

# ======================
# FILE FILTERING
# ======================

--pattern "REGEX"          # Include files matching pattern
--exclude "REGEX"          # Exclude files matching pattern
--context "file1,file2"    # Context files (comma-separated)
--context-pattern "*.md"   # Context file patterns
--recursive                # Process directories recursively
--remove-comments          # Strip comments before processing

# Examples:
--pattern ".*\.(js|jsx|ts|tsx)$"
--exclude ".*\.(test|spec)\.js$"
--context "package.json,README.md,*.config.js"

# ======================
# AI CONFIGURATION
# ======================

--prompt "TEXT"            # AI instruction prompt
--prompt-file PATH         # File containing prompt
--model MODEL_NAME         # Override default model
--temperature 0.1          # Creativity level (0.0-2.0)
--max-tokens 4000          # Maximum response tokens

# ======================
# PROCESSING OPTIONS
# ======================

--concurrent 3             # Max parallel file processing
--dry-run                  # Preview without changes
--verbose                  # Detailed output
--backup                   # Force backup creation
--smart-suffix             # Insert suffix before extension
--suffix TEXT              # Custom suffix text

# ======================
# SAFETY FEATURES
# ======================

# Default behavior is safe
presto --cmd modernize --input legacy.js
# Automatically creates legacy.js.backup

# Preview changes first
presto --cmd optimize --input critical.py --preview

# Create parallel structure for comparison
presto --cmd enhance --input ./src --output directory --output-dir ./enhanced

# ======================
# ENVIRONMENT VARIABLES
# ======================

export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"
export PRESTO_API_KEY="your-key"      # Generic
export PRESTO_BASE_URL="custom-url"   # Custom API endpoint
export PRESTO_MODEL="model-name"      # Default model

# ======================
# COMMON WORKFLOWS
# ======================

# Safe documentation addition
presto --cmd add-docs --input . --recursive

# Modernize with comparison
presto --cmd modernize --input ./legacy --output directory --output-dir ./modernized

# Generate tests with smart naming
presto --prompt "Create unit tests" --input ./src --pattern ".*\.go$" --output separate --smart-suffix --suffix _test

# Preview critical changes
presto --cmd optimize --input ./payment-service --preview

# Custom documentation command
presto --prompt "Add JSDoc" --pattern ".*\.js$" --smart-suffix --suffix .documented --save-command js-docs

# ======================
# SUPPORTED FILE TYPES
# ======================

# ✅ SUPPORTED (Text files)
*.js, *.py, *.go, *.java, *.cpp, *.c, *.rs
*.html, *.css, *.scss, *.less
*.md, *.txt, *.yaml, *.yml, *.json, *.xml
*.sql, *.sh, *.bash, *.ps1
*.dockerfile, *.gitignore, *.env

# ❌ NOT SUPPORTED (Binary files)
*.xlsx, *.docx, *.pdf
*.jpg, *.png, *.gif, *.svg
*.exe, *.bin, *.dll, *.so
*.zip, *.tar, *.gz
```

---

## 🤝 Contributing

We welcome contributions! Whether it's:

- 🐛 Bug fixes
- ✨ New features
- 📚 Documentation improvements
- 🎭 New built-in commands
- 🎯 Provider integrations

Check out [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📜 License

MIT License - see [LICENSE](LICENSE) file for details.
