# Router Integration Examples

This directory contains examples showing how to integrate EBO's HTTP retry middleware with popular Go routers.

## Routers Covered

1. **Chi Router** - A lightweight, idiomatic and composable router
2. **RouteGroup** - A middleware-oriented router from go-pkgz

## Running the Examples

First, install dependencies:

```bash
go mod download
```

Then run the example:

```bash
go run main.go
```

This will start three servers:
- Chi router on port 8080
- RouteGroup on port 8081  
- Advanced configuration on port 8082

## Testing the Endpoints

### Chi Router (port 8080)

```bash
# Health check (no retry)
curl http://localhost:8080/health

# Unstable endpoint (will retry on 5xx)
curl http://localhost:8080/api/unstable

# Rate limited endpoint (will retry on 429)
curl http://localhost:8080/api/rate-limited

# Custom retry logic
curl http://localhost:8080/api/custom
```

### RouteGroup (port 8081)

```bash
# Health check (no retry)
curl http://localhost:8081/health

# Unstable endpoint (will retry)
curl http://localhost:8081/api/unstable

# Rate limited endpoint (will retry)
curl http://localhost:8081/api/rate-limited

# v2 API with additional headers
curl http://localhost:8081/api/v2/data
```

### Advanced Configuration (port 8082)

```bash
# Fast retry for user endpoints
curl http://localhost:8082/api/user/123
curl -X POST http://localhost:8082/api/user

# Aggressive retry for background jobs
curl -X POST http://localhost:8082/api/jobs/process
curl -X POST http://localhost:8082/api/jobs/batch

# External API retry configuration
curl http://localhost:8082/api/external/weather
curl http://localhost:8082/api/external/stocks
```

## Key Patterns

### 1. Chi Router Integration

```go
// Create the middleware
retryMiddleware := ebo.Middleware(ebo.DefaultResponseChecker,
    ebo.Initial(200*time.Millisecond),
    ebo.Tries(3),
    ebo.Jitter(0.3),
)

// Apply to a route group
r.Group(func(r chi.Router) {
    r.Use(retryMiddleware)
    r.Get("/api/endpoint", handler)
})

// Apply to specific routes
r.With(retryMiddleware).Get("/api/endpoint", handler)
```

### 2. RouteGroup Integration

```go
// Create the middleware
retryMiddleware := ebo.Middleware(ebo.DefaultResponseChecker,
    ebo.Initial(100*time.Millisecond),
    ebo.Tries(5),
)

// Apply to a group
apiGroup := router.Group()
apiGroup.Use(retryMiddleware)
apiGroup.HandleFunc("GET /api/endpoint", handler)
```

### 3. Custom Response Checkers

```go
// Create custom checker
customChecker := func(resp *http.Response) bool {
    return resp.StatusCode >= 500 || 
           resp.StatusCode == 429 || 
           resp.StatusCode == 502
}

// Use with middleware
middleware := ebo.Middleware(customChecker, ebo.API())
```

### 4. Different Strategies for Different Endpoints

The advanced example shows how to apply different retry strategies:
- Fast retry for user-facing endpoints (low latency)
- Aggressive retry for background jobs (high reliability)
- Custom retry for external APIs (handle gateway errors)

## Benefits

1. **Automatic Retry**: Failed requests are automatically retried with exponential backoff
2. **Configurable**: Different retry strategies for different endpoints
3. **Router Agnostic**: Works with any router that supports standard middleware
4. **Production Ready**: Includes jitter to prevent thundering herd
5. **Observability**: Easy to add logging or metrics to retry middleware