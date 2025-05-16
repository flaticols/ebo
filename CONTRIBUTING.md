# Contributing to EBO

Thank you for considering contributing to EBO! This guide will help you get started.

## Code of Conduct

Please be respectful and constructive in all interactions.

## How to Contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Commit your changes (see Commit Guidelines below)
5. Push to your branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## Commit Guidelines

We use [Conventional Commits](https://www.conventionalcommits.org/) for our commit messages. This helps us automatically generate changelogs and release notes.

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that do not affect the meaning of the code (formatting, etc)
- **refactor**: A code change that neither fixes a bug nor adds a feature
- **test**: Adding missing tests or correcting existing tests
- **chore**: Changes to the build process or auxiliary tools

### Examples

```
feat(retry): add circuit breaker support

Add support for circuit breaker pattern to prevent
cascading failures when services are down.

Closes #42
```

```
fix(helpers): handle nil response correctly

Previously, a nil response would cause a panic.
This change adds proper nil checking.
```

## Testing

Before submitting a PR, ensure:

1. All tests pass: `go test ./...`
2. Code is linted: `golangci-lint run ./...`
3. New features have appropriate tests
4. Documentation is updated if needed

## Pull Request Process

1. Update the README.md with details of changes to the interface, if applicable
2. Add tests for new functionality
3. Ensure all CI checks pass
4. PR will be merged after review and approval

## Development Setup

1. Clone the repository
2. Install Go 1.23 or higher
3. Install golangci-lint: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
4. Run tests: `go test ./...`

## Questions?

Feel free to open an issue for any questions or discussions.