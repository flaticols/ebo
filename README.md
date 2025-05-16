# EBO

Simple, fast exponential backoff retry for Go.

## Features

- Zero dependencies
- Fast and lightweight
- Functional options for flexible configuration
- Jitter support to prevent thundering herd
- Timeout and retry limit controls
- Sensible defaults

## Installation

```bash
go get github.com/yourusername/ebo
```

## Usage

### Simple retry with defaults

```go
err := ebo.Retry(func() error {
    return doSomething()
})
```

### Quick retry for simple cases

```go
err := ebo.QuickRetry(func() error {
    return doSomething()
})
```

### Custom configuration

```go
err := ebo.Retry(func() error {
    return doSomething()
},
    ebo.WithMaxRetries(5),
    ebo.WithInitialInterval(1*time.Second),
    ebo.WithMaxInterval(30*time.Second),
    ebo.WithRandomizeFactor(0.5),
)
```

### Time-based retry

```go
err := ebo.Retry(func() error {
    return doSomething()
},
    ebo.WithMaxElapsedTime(5*time.Minute),
    ebo.WithMaxRetries(0), // No retry limit, only time limit
)
```

### Basic backoff without options

```go
err := ebo.RetryWithBackoff(func() error {
    return doSomething()
}, 3) // max 3 retries
```

## Options

- `WithInitialInterval(d)` - Set initial retry interval
- `WithMaxInterval(d)` - Set maximum retry interval
- `WithMaxRetries(n)` - Set maximum retry attempts (0 for no limit)
- `WithMultiplier(f)` - Set backoff multiplier (default 2.0)
- `WithMaxElapsedTime(d)` - Set maximum total time for retries
- `WithRandomizeFactor(f)` - Set jitter factor (0-1, default 0.5)

## Default Configuration

```go
InitialInterval: 500ms
MaxInterval:     30s
MaxRetries:      10
Multiplier:      2.0
MaxElapsedTime:  5m
RandomizeFactor: 0.5
```

## License

MIT