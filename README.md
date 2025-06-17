# Presto

**AI-Powered File Processor** - Transform, generate, and enhance your codebase using AI

Presto streamlines your development workflow by applying AI transformations to files and directories. Whether you're adding documentation, modernizing code, or generating new content, Presto makes it simple and consistent.

## 🎯 What Presto Does

- **Transform existing files** with AI-powered modifications
- **Generate new content** from context and prompts
- **Batch process** entire directories with intelligent filtering
- **Preserve your workflow** with flexible output modes and backups
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
# Add documentation to a single file
presto --prompt "Add comprehensive documentation" --input main.go

# Use a predefined command on entire project
presto --cmd add-docs --input . --recursive

# Generate a README from your codebase
presto --cmd summarize --input . --context "*.go,*.md" --output-file README.md
```

## 🎪 Real-World Examples

### 1. Legacy Code Modernization

Transform an old JavaScript project to modern ES6+:

```bash
# Modernize all JavaScript files
presto --cmd modernize \
  --input ./src \
  --recursive \
  --pattern ".*\.(js|jsx)$" \
  --output separate \
  --suffix .modern.js

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

**After (utils.js.modern):**

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

Document an entire Go microservice:

```bash
# Add docs to all Go files with context from README
presto --cmd add-docs \
  --input ./internal \
  --recursive \
  --context README.md,go.mod \
  --backup
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

**After:**

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
//
// Example:
//   user, err := service.GetUser("user-123")
//   if err != nil {
//       log.Printf("Failed to get user: %v", err)
//       return
//   }
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

### 3. Adding Logging and Error Handling

Enhance a Python Flask API with proper logging:

```bash
# Add comprehensive logging to API endpoints
presto --prompt "Add structured logging for debugging and monitoring. Include request IDs, timing, and proper error handling." \
  --input app.py \
  --output inplace \
  --backup
```

**Before:**

```python
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route('/users/<user_id>')
def get_user(user_id):
    user = database.get_user(user_id)
    if not user:
        return jsonify({'error': 'Not found'}), 404
    return jsonify(user.to_dict())
```

**After:**

```python
import logging
import time
import uuid
from flask import Flask, request, jsonify, g

app = Flask(__name__)

# Configure structured logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(name)s: %(message)s'
)
logger = logging.getLogger(__name__)

@app.before_request
def before_request():
    g.request_id = str(uuid.uuid4())[:8]
    g.start_time = time.time()
    logger.info(f"[{g.request_id}] Starting {request.method} {request.path}")

@app.after_request
def after_request(response):
    duration = time.time() - g.start_time
    logger.info(f"[{g.request_id}] Completed {request.method} {request.path} "
               f"- {response.status_code} in {duration:.2f}s")
    return response

@app.route('/users/<user_id>')
def get_user(user_id):
    """Retrieve user by ID with comprehensive logging and error handling."""
    try:
        logger.info(f"[{g.request_id}] Fetching user: {user_id}")

        user = database.get_user(user_id)
        if not user:
            logger.warning(f"[{g.request_id}] User not found: {user_id}")
            return jsonify({
                'error': 'User not found',
                'request_id': g.request_id
            }), 404

        logger.info(f"[{g.request_id}] Successfully retrieved user: {user_id}")
        return jsonify({
            'data': user.to_dict(),
            'request_id': g.request_id
        })

    except database.DatabaseError as e:
        logger.error(f"[{g.request_id}] Database error for user {user_id}: {str(e)}")
        return jsonify({
            'error': 'Internal server error',
            'request_id': g.request_id
        }), 500

    except Exception as e:
        logger.exception(f"[{g.request_id}] Unexpected error for user {user_id}: {str(e)}")
        return jsonify({
            'error': 'Internal server error',
            'request_id': g.request_id
        }), 500
```

### 4. Generating Project Documentation

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

### 5. Code Review and Optimization

Optimize performance-critical code:

```bash
# Analyze and optimize database queries
presto --prompt "Optimize these database queries for performance. Add indexing suggestions and explain the improvements." \
  --input ./repositories \
  --pattern ".*\.go$" \
  --context "schema.sql" \
  --output separate \
  --suffix .optimized
```

### 6. Testing and Quality Assurance

Generate comprehensive tests:

```bash
# Generate unit tests for all services
presto --prompt "Generate comprehensive unit tests with mocking, edge cases, and good coverage" \
  --input ./services \
  --pattern ".*\.go$" \
  --exclude ".*_test\.go$" \
  --output separate \
  --suffix _test.go
```

### 7. Multi-language Documentation Translation

Convert documentation to different formats:

```bash
# Convert Go comments to JSDoc format for TypeScript migration
presto --prompt "Convert Go-style comments to JSDoc format for TypeScript" \
  --input ./api-client.go \
  --var TARGET_FORMAT=typescript \
  --output-file api-client.ts
```

## 🎛️ Command Reference

### Output Modes

- **`--output inplace`** - Modify original files (creates backups with `--backup`)
- **`--output separate`** - Create new files with suffix (default: `.presto`)
- **`--output stdout`** - Print results to terminal
- **`--output file`** - Single output file (for generate mode)

### File Filtering

```bash
# Process specific file types
--pattern ".*\.(js|jsx|ts|tsx)$"

# Skip certain files
--exclude ".*\.(test|spec)\.js$"

# Process recursively
--recursive

# Include context files
--context "package.json,README.md,*.config.js"
```

### AI Configuration

```bash
# Different AI providers
presto configure  # Interactive setup

# Override model for specific tasks
--model gpt-4-turbo          # More capable for complex tasks
--model gpt-3.5-turbo        # Faster and cheaper
--model claude-3-5-sonnet    # Anthropic's latest

# Adjust creativity
--temperature 0.1  # More focused and deterministic
--temperature 0.7  # More creative and varied
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
  output_mode: "separate"
  output_suffix: ".presto"
  backup_original: true
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
```

## 🎪 Creating Custom Commands

### Save Current Options as Command

```bash
# Create a custom documentation command
presto --prompt "Add JSDoc comments with TypeScript types" \
  --pattern ".*\.(js|jsx)$" \
  --output separate \
  --suffix .documented \
  --save-command js-docs

# Use your custom command
presto --cmd js-docs --input ./src --recursive
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
  --output separate \
  --suffix .autodoc

# Check for outdated documentation
presto --prompt "Compare this code with its documentation and flag any inconsistencies" \
  --input ./src \
  --context "*.md" \
  --output file \
  --output-file doc-analysis.md
```

### Team Consistency

```bash
# Standardize code style across team
presto --prompt "Ensure this code follows our team's style guide" \
  --input . \
  --recursive \
  --context ".eslintrc,.prettierrc,STYLE_GUIDE.md" \
  --output inplace \
  --backup
```

### Migration Assistance

```bash
# Migrate from Vue 2 to Vue 3
presto --prompt "Migrate this Vue 2 component to Vue 3 Composition API" \
  --input ./components \
  --pattern ".*\.vue$" \
  --context "package.json,migration-notes.md" \
  --output separate \
  --suffix .vue3
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

### 1. Start Small

```bash
# Test on a single file first
presto --prompt "Add comments" --input example.js --dry-run
```

### 2. Use Context Wisely

```bash
# Include relevant context for better results
presto --cmd add-docs \
  --input ./api \
  --context "README.md,package.json,api-spec.yaml"
```

### 3. Backup Important Files

```bash
# Always backup when modifying in place
presto --output inplace --backup
```

### 4. Leverage Dry Run

```bash
# Preview changes before applying
presto --cmd modernize --input ./legacy --dry-run --verbose
```

### 5. Custom Prompts for Specific Needs

```bash
# Be specific about requirements
presto --prompt "Add error handling following our company's error handling guidelines. Include structured logging with correlation IDs." \
  --context "error-guidelines.md" \
  --input ./handlers
```

### 6. Optimize for Large Codebases

```bash
# Process in batches for large projects
presto --cmd add-docs \
  --input ./src \
  --pattern ".*\.(go|js)$" \
  --concurrent 5 \
  --max-tokens 2000
```

## 🚀 Performance Tips

- **Use file patterns** to avoid processing unnecessary files
- **Adjust concurrency** based on your system and API limits
- **Set reasonable token limits** to control costs
- **Use context selectively** - too much context can confuse the AI
- **Cache results** by using separate output mode for iterative improvements

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
# PRESTO COMMAND REFERENCE

# ======================
# BASIC USAGE
# ======================

# Interactive setup
presto configure

# Basic file transformation
presto --prompt "Your instruction" --input file.js
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
# INPUT/OUTPUT OPTIONS
# ======================

--input PATH               # File or directory to process
--output MODE              # inplace|separate|stdout|file
--output-file PATH         # Output file path (generate mode)
--suffix SUFFIX            # Suffix for separate mode (default: .presto)
--recursive                # Process directories recursively
--backup                   # Create .backup files (with inplace mode)

# ======================
# FILE FILTERING
# ======================

--pattern "REGEX"          # Include files matching pattern
--exclude "REGEX"          # Exclude files matching pattern
--context "file1,file2"    # Context files (comma-separated)
--context-pattern "*.md"   # Context file patterns
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

# Models:
# OpenAI: gpt-4, gpt-4-turbo, gpt-3.5-turbo
# Anthropic: claude-3-5-sonnet, claude-3-haiku
# Local: depends on your setup

# ======================
# PROCESSING OPTIONS
# ======================

--concurrent 3             # Max parallel file processing
--dry-run                  # Preview without changes
--verbose                  # Detailed output
--generate                 # Generate mode (vs transform)

# ======================
# CUSTOM COMMANDS
# ======================

# Save current options as reusable command
--save-command NAME

# Use variables in commands
--var "KEY=value,KEY2=value2"

# Manage custom commands
--list-commands            # Show all available commands
--show-command NAME        # Show command details
--delete-command NAME      # Remove custom command

# ======================
# ENVIRONMENT VARIABLES
# ======================

export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"
export OPENROUTER_API_KEY="your-key"  # Legacy support
export PRESTO_API_KEY="your-key"      # Generic
export PRESTO_BASE_URL="custom-url"   # Custom API endpoint
export PRESTO_MODEL="model-name"      # Default model

# ======================
# COMMON WORKFLOWS
# ======================

# Add docs to entire project
presto --cmd add-docs --input . --recursive --backup

# Modernize JavaScript files
presto --cmd modernize --input ./src --pattern ".*\.js$" --output separate

# Generate README from codebase
presto --cmd summarize --context "*.go,*.md" --output-file README.md

# Add logging to API handlers
presto --cmd add-logging --input ./handlers --recursive --output inplace --backup

# Convert code format with variables
presto --cmd convert --var "TARGET_FORMAT=typescript" --input api.js

# Clean up legacy code
presto --cmd cleanup --input ./legacy --exclude ".*\.min\.js$" --recursive

# Optimize database queries
presto --prompt "Optimize SQL performance" --input ./queries --context "schema.sql"

# Custom documentation with context
presto --prompt "Add JSDoc with TypeScript types" --input ./src --context "types.d.ts" --pattern ".*\.js$" --save-command js-typed-docs

# ======================
# HELP COMMANDS
# ======================

presto --help             # Show basic help
presto --version          # Show version
presto --list-commands    # List all commands
presto --show-command NAME # Show command details

# ======================
# EXAMPLES BY FILE TYPE
# ======================

# JavaScript/TypeScript
presto --cmd modernize --input . --pattern ".*\.(js|jsx|ts|tsx)$"

# Python
presto --cmd add-logging --input . --pattern ".*\.py$" --exclude "__pycache__"

# Go
presto --cmd add-docs --input . --pattern ".*\.go$" --exclude ".*_test\.go$"

# SQL
presto --prompt "Add comments and optimize" --input . --pattern ".*\.sql$"

# Multiple languages
presto --cmd cleanup --input . --pattern "\.(js|py|go|java)$" --recursive

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
