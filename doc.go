// Package ebo provides a simple, fast exponential backoff retry library for Go.
//
// The library offers three levels of API complexity to suit different needs:
//
// 1. Basic retry with sensible defaults:
//
//	err := ebo.Retry(func() error {
//	    return doSomething()
//	})
//
// 2. Configured retry with short options:
//
//	err := ebo.Retry(func() error {
//	    return doSomething()
//	}, ebo.Tries(5), ebo.Initial(1*time.Second))
//
// 3. Iterator pattern for complex retry scenarios:
//
//	for attempt := range ebo.Attempts(ebo.Tries(3)) {
//	    if err := doWork(); err == nil {
//	        break
//	    }
//	    log.Printf("Attempt %d failed", attempt.Number)
//	}
//
// The library includes convenient presets for common use cases:
//
//	// Quick retries for fast operations
//	err := ebo.Retry(checkCache, ebo.Quick())
//
//	// API calls with moderate retry
//	err := ebo.Retry(callAPI, ebo.API())
//
//	// Database operations with longer timeouts
//	err := ebo.Retry(connectDB, ebo.Database())
//
//	// HTTP requests with status code awareness
//	client := ebo.NewHTTPClient(ebo.HTTPStatus())
//
// Advanced features include:
//   - Context-aware retries
//   - Custom retry conditions
//   - Retry with logging
//   - HTTP client integration
//   - Permanent error handling
//   - Iterator pattern for stateful retries (using Go 1.23+ iter package)
//
// For more examples and documentation, visit https://github.com/flaticols/ebo
package ebo
