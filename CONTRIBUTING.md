# Contributing to Bifrost

Thank you for your interest in contributing to Bifrost! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Commit Convention](#commit-convention)
- [Pull Request Process](#pull-request-process)
- [Testing](#testing)
- [Code Style](#code-style)

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow
- Follow the project's technical guidelines

## How Can I Contribute?

### Reporting Bugs

Before creating a bug report:
- Check existing issues to avoid duplicates
- Use the latest version of Bifrost
- Collect relevant information (OS, Go version, error logs)

**Bug Report Template:**
```markdown
**Describe the bug**
A clear description of what the bug is.

**To Reproduce**
Steps to reproduce:
1. Go to '...'
2. Click on '...'
3. See error

**Expected behavior**
What you expected to happen.

**Environment:**
- OS: [e.g., macOS 14.0, Ubuntu 22.04]
- Bifrost version: [e.g., v0.0.3]
- Go version: [e.g., 1.21.5]
```

### Suggesting Features

Feature requests are welcome! Please:
- Check if the feature already exists or is planned
- Explain the use case clearly
- Consider the scope (should it be in core or a plugin?)

### Contributing Code

1. Fork the repository
2. Create a feature branch from `develop`
3. Make your changes
4. Write tests if applicable
5. Submit a Pull Request

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Git
- A terminal with Unicode support

### Clone and Build

```bash
# Fork and clone your fork
git clone https://github.com/YOUR_USERNAME/bifrost.git
cd bifrost

# Add upstream remote
git remote add upstream https://github.com/steevenmentech/bifrost.git

# Install dependencies
go mod download

# Build
go build -o bifrost ./cmd/bifrost

# Run
./bifrost
```

### Development Workflow

```bash
# Create a feature branch
git checkout -b feature/my-feature develop

# Make changes and test
go build ./...
go test ./...

# Commit using conventional commits
git commit -m "feat: Add new feature"

# Push to your fork
git push origin feature/my-feature

# Create Pull Request on GitHub
```

## Project Structure

```
bifrost/
├── cmd/
│   └── bifrost/          # Application entry point
├── internal/             # Private application code
│   ├── config/           # Configuration management
│   ├── keyring/          # Secure credential storage
│   ├── sftp/             # SFTP client implementation
│   ├── ssh/              # SSH client implementation
│   └── tui/              # Terminal UI
│       ├── keys/         # Keybinding definitions
│       ├── styles/       # UI styles and themes
│       └── views/        # UI components (forms, browsers, etc.)
├── pkg/                  # Public reusable packages
├── assets/               # Logo and images
└── configs/              # Configuration file templates
```

### Key Components

| Component | Purpose |
|-----------|---------|
| `cmd/bifrost/main.go` | Application entry point and CLI handling |
| `internal/tui/` | All Terminal UI logic using Bubble Tea |
| `internal/sftp/` | SFTP operations (browse, download, edit) |
| `internal/ssh/` | SSH terminal session management |
| `internal/config/` | Connection storage and configuration |
| `internal/keyring/` | Secure password storage |

## Commit Convention

We use [Conventional Commits](https://www.conventionalcommits.org/) for clear changelog generation.

### Format

```
<type>(<scope>): <description>

[optional body]
```

### Types

| Type | Use Case | Appears in Changelog |
|------|----------|---------------------|
| `feat:` | New feature | ✓ |
| `fix:` | Bug fix | ✓ |
| `docs:` | Documentation only | ✗ |
| `style:` | Code formatting (no logic change) | ✗ |
| `refactor:` | Code refactoring | ✗ |
| `perf:` | Performance improvement | ✓ |
| `test:` | Adding or updating tests | ✗ |
| `chore:` | Maintenance tasks | ✗ |
| `ci:` | CI/CD changes | ✗ |

### Examples

```bash
feat(sftp): Add file upload functionality
fix(ssh): Fix connection timeout on slow networks
docs: Update README installation instructions
refactor(tui): Simplify error handling
test: Add tests for SFTP download
ci: Add automated release workflow
```

### Scope (optional)

- `sftp` - SFTP-related changes
- `ssh` - SSH-related changes
- `tui` - UI-related changes
- `config` - Configuration changes
- `ci` - CI/CD changes

## Pull Request Process

1. **Update your branch**
   ```bash
   git fetch upstream
   git rebase upstream/develop
   ```

2. **Run tests and build**
   ```bash
   go test ./...
   go build ./...
   ```

3. **Create Pull Request**
   - Use a clear title following commit conventions
   - Fill out the PR template
   - Link related issues
   - Request review

4. **PR Review**
   - Address review comments
   - Keep commits clean (squash if needed)
   - Ensure CI passes

5. **Merge**
   - PRs are merged by maintainers
   - Squash merge for feature branches
   - Regular merge for release branches

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/sftp/...

# Verbose output
go test -v ./...
```

### Writing Tests

- Place tests in `*_test.go` files
- Use table-driven tests when possible
- Mock external dependencies (SSH, SFTP connections)
- Test both success and error cases

Example:
```go
func TestSFTPDownload(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        wantErr bool
    }{
        {"valid file", "/path/to/file.txt", false},
        {"missing file", "/nonexistent", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic here
        })
    }
}
```

## Code Style

### Go Standards

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting (automatically applied by most editors)
- Run `go vet` to catch common mistakes
- Use meaningful variable names

### Best Practices

1. **Error Handling**
   ```go
   // Good
   if err != nil {
       return fmt.Errorf("failed to connect: %w", err)
   }

   // Bad
   if err != nil {
       panic(err)
   }
   ```

2. **Naming Conventions**
   - Use camelCase for unexported names
   - Use PascalCase for exported names
   - Use descriptive names (avoid single letters except in loops)

3. **Comments**
   - Add package comments
   - Document exported functions/types
   - Explain complex logic with inline comments

4. **Keep Functions Small**
   - Single responsibility principle
   - Extract complex logic into helper functions

### UI/UX Guidelines

- Follow existing keyboard shortcuts (vim-style when possible)
- Maintain consistent styling with lipgloss
- Add help text for new features
- Test on different terminal sizes

## Questions?

- Open an issue for questions
- Check existing issues and PRs
- Review the README for basic information

## License

By contributing, you agree that your contributions will be licensed under the Non-Commercial License.

---

Thank you for contributing to Bifrost! 🌈
