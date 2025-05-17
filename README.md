# EBO

[![Lint & Tests](https://github.com/flaticols/ebo/actions/workflows/ci.yml/badge.svg)](https://github.com/flaticols/ebo/actions/workflows/ci.yml)

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
go get github.com/flaticols/ebo
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
// New short API
err := ebo.Retry(func() error {
    return doSomething()
},
    ebo.Tries(5),
    ebo.Initial(1*time.Second),
    ebo.Max(30*time.Second),
    ebo.Jitter(0.5),
)

// Using presets
err := ebo.Retry(func() error {
    return apiCall()
}, ebo.API())  // Sensible API defaults

err := ebo.Retry(func() error {
    return dbConnect()
}, ebo.Database())  // Database-optimized settings
```

### Time-based retry

```go
err := ebo.Retry(func() error {
    return doSomething()
}, ebo.Timeout(5*time.Minute))  // Only time-based retry
```

### Basic backoff without options

```go
err := ebo.RetryWithBackoff(func() error {
    return doSomething()
}, 3) // max 3 retries
```

### Context-aware retry

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := ebo.RetryWithContext(ctx, func() error {
    return doSomething()
},
    ebo.Initial(1*time.Second),
    ebo.Tries(10),
)
```

### Retry with logging

```go
logger := log.New(os.Stdout, "[RETRY] ", log.LstdFlags)

err := ebo.RetryWithLogging(func() error {
    return doSomething()
}, logger, ebo.Quick())
```

### Custom retry conditions

```go
// Define what errors are retryable
isRetryable := func(err error) bool {
    if err == nil {
        return false
    }
    // Don't retry context cancellations
    if errors.Is(err, context.Canceled) {
        return false
    }
    // Don't retry specific business errors
    if errors.Is(err, ErrUserNotFound) {
        return false
    }
    return true
}

err := ebo.RetryWithCondition(func() error {
    return doSomething()
}, isRetryable,
    ebo.WithMaxRetries(3),
)
```

### HTTP client with retry

```go
// Create an HTTP client with built-in retry
client := ebo.NewHTTPClient(ebo.HTTPStatus())

resp, err := client.Get("https://api.example.com/data")
```

### HTTP request with retry

```go
req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)

resp, err := ebo.HTTPDo(req, http.DefaultClient, ebo.API())
```

### HTTP Middleware

EBO provides HTTP middleware that automatically retries requests based on response codes.

```go
// Create a handler
handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Your API logic here
})

// Wrap with retry middleware (retries on 5xx and 429 by default)
retryHandler := ebo.NewRetryMiddleware(handler, ebo.DefaultResponseChecker,
    ebo.Initial(500*time.Millisecond),
    ebo.Tries(5),
    ebo.Jitter(0.3),
)

// Use with standard HTTP server
http.ListenAndServe(":8080", retryHandler)

// Or use the middleware function for router compatibility
middleware := ebo.Middleware(ebo.DefaultResponseChecker, ebo.API())
http.Handle("/api/", middleware(handler))

// Custom response checker
customChecker := func(resp *http.Response) bool {
    return resp.StatusCode >= 500 || resp.StatusCode == 404
}
customMiddleware := ebo.Middleware(customChecker, ebo.Quick())
```

### Router Integration

EBO's middleware works seamlessly with popular Go routers:

```go
// Chi router
import "github.com/go-chi/chi/v5"

r := chi.NewRouter()
r.Use(ebo.Middleware(ebo.DefaultResponseChecker, ebo.API()))
r.Get("/api/users", usersHandler)

// RouteGroup
import "github.com/go-pkgz/routegroup"

router := routegroup.New(http.NewServeMux())
apiGroup := router.Group()
apiGroup.Use(ebo.Middleware(ebo.DefaultResponseChecker, ebo.Quick()))
apiGroup.HandleFunc("GET /api/data", dataHandler)
```

See [examples/router-integration](examples/router-integration) for complete examples with chi and routegroup.

## Iterator Pattern (Go 1.23+)

EBO now supports the new Go iterator pattern for more flexible and elegant retry loops.

### When to Use the Iterator Pattern

The iterator pattern is ideal when you need:
- **Fine-grained control** over retry logic (custom success/failure conditions)
- **Stateful retries** (tracking attempts, partial successes, etc.)
- **Complex retry patterns** (circuit breakers, hedged requests, progressive fallbacks)
- **Custom backoff sequences** or non-standard retry timing
- **Integration with existing control flow** (select statements, goroutines)

### Basic iterator usage

```go
for attempt := range ebo.Attempts(
    ebo.Tries(3),
    ebo.Initial(1*time.Second),
) {
    fmt.Printf("Attempt %d (delay: %v)\n", attempt.Number, attempt.Delay)
    
    result, err := doSomething()
    if err == nil {
        return result, nil
    }
    
    // Custom retry logic
    if !isRetryable(err) {
        return nil, err // Don't retry certain errors
    }
    
    attempt.LastError = err
}
```

### Context-aware iterator

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

for attempt := range ebo.AttemptsWithContext(ctx,
    ebo.Tries(10),
) {
    result, err := doWork(attempt.Context)
    if err == nil {
        return result, nil
    }
    
    if errors.Is(err, ErrServiceUnavailable) {
        time.Sleep(5 * time.Second) // Custom delay for specific errors
    }
}
```

### Use Cases by Pattern

#### Circuit Breaker Pattern
```go
type CircuitBreaker struct {
    failureCount int
    threshold    int
    mu          sync.Mutex
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    if cb.failureCount >= cb.threshold {
        cb.mu.Unlock()
        return ErrCircuitOpen
    }
    cb.mu.Unlock()
    
    for attempt := range ebo.Attempts(ebo.WithMaxRetries(3)) {
        err := fn()
        
        cb.mu.Lock()
        if err == nil {
            cb.failureCount = 0
            cb.mu.Unlock()
            return nil
        }
        
        cb.failureCount++
        cb.mu.Unlock()
    }
    
    return ErrMaxRetriesExceeded
}
```

#### Progressive Fallback Pattern
```go
endpoints := []string{"primary.api.com", "secondary.api.com", "fallback.api.com"}

for i, endpoint := range endpoints {
    for attempt := range ebo.Attempts(ebo.WithMaxRetries(2)) {
        resp, err := callEndpoint(endpoint)
        if err == nil {
            return resp, nil
        }
        
        log.Printf("Endpoint %s attempt %d failed", endpoint, attempt.Number)
    }
    
    log.Printf("Endpoint %s failed, trying next", endpoint)
}

return nil, errors.New("all endpoints failed")
```

#### Hedged Requests Pattern
```go
results := make(chan Result, 3)

// Launch parallel requests with staggered starts
for i, delay := range []time.Duration{0, 500*time.Millisecond, 2*time.Second} {
    go func(id int, startDelay time.Duration) {
        time.Sleep(startDelay)
        
        for attempt := range ebo.Attempts(ebo.WithMaxRetries(2)) {
            result, err := makeRequest(id)
            if err == nil {
                results <- result
                return
            }
        }
    }(i, delay)
}

// Return first successful result
select {
case result := <-results:
    return result, nil
case <-time.After(10 * time.Second):
    return nil, errors.New("timeout waiting for hedged requests")
}
```

### HTTP request with iterator

```go
var response *http.Response

for attempt := range ebo.Attempts(ebo.WithMaxRetries(5)) {
    resp, err := http.Get("https://api.example.com/data")
    if err != nil {
        attempt.LastError = err
        continue
    }
    
    if resp.StatusCode >= 500 {
        resp.Body.Close()
        attempt.LastError = fmt.Errorf("server error: %d", resp.StatusCode)
        continue
    }
    
    response = resp
    break
}
```

### Stateful retry

```go
var successCount int

for attempt := range ebo.Attempts(ebo.WithMaxRetries(5)) {
    if err := doOperation(); err == nil {
        successCount++
        if successCount >= 2 {
            break // Need 2 successful attempts
        }
    }
    
    fmt.Printf("Attempt %d: %d successful so far\n", attempt.Number, successCount)
}
```

### Simple helper function

If you don't need fine control, use the helper functions:

```go
// Simple usage with DoWithAttempts
err := ebo.DoWithAttempts(func(attempt *ebo.Attempt) error {
    log.Printf("Attempt %d", attempt.Number)
    return apiCall()
}, ebo.Tries(3))

// With context
ctx := context.Background()
err := ebo.DoWithAttemptsContext(ctx, func(attempt *ebo.Attempt) error {
    return apiCallWithContext(attempt.Context)
}, ebo.Tries(5))
```

## Options

### Short API (Recommended)

- `Initial(d)` - Set initial retry interval
- `Max(d)` - Set maximum retry interval  
- `Tries(n)` - Set maximum retry attempts (0 for no limit)
- `Multiplier(f)` - Set backoff multiplier
- `Jitter(f)` - Set jitter factor (0-1)
- `MaxTime(d)` - Set maximum total time for retries
- `NoJitter()` - Disable jitter completely
- `Forever()` - No retry limit (only time-based)
- `Linear()` - Constant interval (no exponential backoff)
- `Exponential(f)` - Exponential backoff with custom factor

### Presets

- `Quick()` - Fast retries for quick operations
- `API()` - Optimized for API calls
- `Database()` - Optimized for database operations
- `HTTPStatus()` - Optimized for HTTP status retries
- `Aggressive()` - Fast, many retries
- `Gentle()` - Slow, few retries


## Default Configuration

```go
InitialInterval: 500ms
MaxInterval:     30s
MaxRetries:      10
Multiplier:      2.0
MaxElapsedTime:  5m
RandomizeFactor: 0.5
```

## API Reference

### Core Functions

- `Retry(fn RetryableFunc, opts ...Option) error` - Main retry function with exponential backoff
- `QuickRetry(fn RetryableFunc) error` - Simplified retry with sensible defaults
- `RetryWithBackoff(fn RetryableFunc, maxRetries int) error` - Simple exponential backoff without configuration

### Helper Functions

- `RetryWithContext(ctx context.Context, fn func() error, opts ...Option) error` - Context-aware retry
- `RetryWithLogging(fn func() error, logger *log.Logger, opts ...Option) error` - Retry with logging
- `RetryWithCondition(fn func() error, condition func(error) bool, opts ...Option) error` - Custom retry conditions

### HTTP Helpers

- `NewHTTPClient(opts ...Option) *http.Client` - Create HTTP client with retry capability
- `HTTPDo(req *http.Request, client *http.Client, opts ...Option) (*http.Response, error)` - Execute HTTP request with retry

### Iterator Functions (Go 1.23+)

- `Attempts(opts ...Option) func(func(*Attempt) bool)` - Create a retry iterator
- `AttemptsWithContext(ctx context.Context, opts ...Option) func(func(*Attempt) bool)` - Context-aware iterator
- `DoWithAttempts(fn RetryFunc, opts ...Option) error` - Simple iterator-based retry
- `DoWithAttemptsContext(ctx context.Context, fn RetryFunc, opts ...Option) error` - Context-aware iterator retry

### Types

- `RetryableFunc func() error` - Function signature for retryable operations
- `Option func(*RetryConfig)` - Configuration option function
- `HTTPRetryTransport` - http.RoundTripper implementation with retry logic
- `Attempt` - Retry attempt information for iterators
- `RetryFunc func(*Attempt) error` - Function signature for iterator-based retries

## Common Patterns

### HTTP Client with Retry

```go
type RetryableClient struct {
    client *http.Client
}

func (c *RetryableClient) Do(req *http.Request) (*http.Response, error) {
    var resp *http.Response
    err := ebo.Retry(func() error {
        r, err := c.client.Do(req)
        if err != nil {
            return err
        }
        if r.StatusCode >= 500 || r.StatusCode == 429 {
            r.Body.Close()
            return fmt.Errorf("retryable status: %d", r.StatusCode)
        }
        resp = r
        return nil
    },
        ebo.WithInitialInterval(500*time.Millisecond),
        ebo.WithMaxInterval(10*time.Second),
        ebo.WithMaxRetries(5),
    )
    return resp, err
}
```

### HTTP Middleware

```go
func RetryMiddleware(next http.RoundTripper) http.RoundTripper {
    return http.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
        var resp *http.Response
        err := ebo.Retry(func() error {
            r, err := next.RoundTrip(req)
            if err != nil {
                return err
            }
            if r.StatusCode >= 500 || r.StatusCode == 429 {
                r.Body.Close()
                return fmt.Errorf("retryable status: %d", r.StatusCode)
            }
            resp = r
            return nil
        },
            ebo.WithMaxRetries(3),
            ebo.WithInitialInterval(1*time.Second),
        )
        return resp, err
    })
}

// Usage:
client := &http.Client{
    Transport: RetryMiddleware(http.DefaultTransport),
}
```

### Database Connections

```go
func ConnectWithRetry(dsn string) (*sql.DB, error) {
    var db *sql.DB
    err := ebo.Retry(func() error {
        conn, err := sql.Open("postgres", dsn)
        if err != nil {
            return err
        }
        if err := conn.Ping(); err != nil {
            conn.Close()
            return err
        }
        db = conn
        return nil
    },
        ebo.WithInitialInterval(1*time.Second),
        ebo.WithMaxInterval(30*time.Second),
        ebo.WithMaxElapsedTime(2*time.Minute),
    )
    return db, err
}
```

### Context-Aware Retry

```go
func RetryWithContext(ctx context.Context, fn func() error, opts ...ebo.Option) error {
    return ebo.Retry(func() error {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            return fn()
        }
    }, opts...)
}
```

### Custom Retry Conditions

```go
func IsRetryable(err error) bool {
    if err == nil {
        return false
    }
    if errors.Is(err, io.EOF) {
        return false
    }
    if errors.Is(err, context.Canceled) {
        return false
    }
    var netErr *net.Error
    if errors.As(err, &netErr) && netErr.Timeout() {
        return true
    }
    return true
}

func RetryWithCondition(fn func() error, condition func(error) bool) error {
    return ebo.Retry(func() error {
        err := fn()
        if err != nil && !condition(err) {
            return fmt.Errorf("non-retryable error: %w", err)
        }
        return err
    })
}
```

## Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to contribute to this project.

## License

MIT