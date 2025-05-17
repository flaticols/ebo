package ebo

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

// RetryWithContext respects context cancellation during retries.
// The retry will stop immediately if the context is cancelled.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	err := ebo.RetryWithContext(ctx, func() error {
//	    return performLongOperation()
//	}, ebo.Tries(10), ebo.Initial(1*time.Second))
func RetryWithContext(ctx context.Context, fn func() error, opts ...Option) error {
	return Retry(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fn()
		}
	}, opts...)
}

// RetryWithLogging adds logging to track retry attempts.
// Each failed attempt will be logged with the error details.
//
// Example:
//
//	logger := log.New(os.Stdout, "[RETRY] ", log.LstdFlags)
//
//	err := ebo.RetryWithLogging(func() error {
//	    return connectToDatabase()
//	}, logger, ebo.Tries(5), ebo.Initial(1*time.Second))
func RetryWithLogging(fn func() error, logger *log.Logger, opts ...Option) error {
	attempt := 0
	return Retry(func() error {
		attempt++
		err := fn()
		if err != nil {
			logger.Printf("Attempt %d failed: %v", attempt, err)
		}
		return err
	}, opts...)
}

// RetryWithCondition allows custom retry conditions.
// Only errors that satisfy the condition function will be retried.
//
// Example:
//
//	isRetryable := func(err error) bool {
//	    if err == nil {
//	        return false
//	    }
//	    // Don't retry authentication errors
//	    if errors.Is(err, ErrAuthFailed) {
//	        return false
//	    }
//	    return true
//	}
//
//	err := ebo.RetryWithCondition(func() error {
//	    return callAPI()
//	}, isRetryable, ebo.Tries(3))
func RetryWithCondition(fn func() error, condition func(error) bool, opts ...Option) error {
	return Retry(func() error {
		err := fn()
		if err != nil && !condition(err) {
			// Return a special error type that won't be retried
			return &permanentError{err}
		}
		return err
	}, opts...)
}

// permanentError wraps an error to indicate it should not be retried.
type permanentError struct {
	err error
}

func (e *permanentError) Error() string {
	return fmt.Sprintf("non-retryable error: %v", e.err)
}

func (e *permanentError) Unwrap() error {
	return e.err
}

// HTTPRetryTransport implements http.RoundTripper with retry logic
type HTTPRetryTransport struct {
	Transport http.RoundTripper
	Options   []Option
}

// RoundTrip implements the http.RoundTripper interface
func (t *HTTPRetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	var resp *http.Response
	err := Retry(func() error {
		r, err := transport.RoundTrip(req)
		if err != nil {
			return err
		}
		resp = r

		// Check if the status code is retryable
		if r.StatusCode >= 500 || r.StatusCode == 429 {
			_ = r.Body.Close()
			return fmt.Errorf("retryable status: %d", r.StatusCode)
		}

		return nil
	}, t.Options...)

	return resp, err
}

// NewHTTPClient creates an HTTP client with retry capabilities.
// The client will automatically retry failed requests based on the provided options.
//
// Example:
//
//	client := ebo.NewHTTPClient(ebo.HTTPStatus())
//
//	resp, err := client.Get("https://api.example.com/data")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer resp.Body.Close()
func NewHTTPClient(opts ...Option) *http.Client {
	return &http.Client{
		Transport: &HTTPRetryTransport{
			Transport: http.DefaultTransport,
			Options:   opts,
		},
	}
}

// RetryableHTTPFunc is a function that can be retried for HTTP requests
type RetryableHTTPFunc func(*http.Request) (*http.Response, error)

// HTTPDo wraps an HTTP request with retry logic.
// It will retry the request based on the response status code and the provided options.
//
// Example:
//
//	req, _ := http.NewRequest("POST", "https://api.example.com/data", body)
//	req.Header.Set("Content-Type", "application/json")
//
//	resp, err := ebo.HTTPDo(req, nil, ebo.API())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer resp.Body.Close()
func HTTPDo(req *http.Request, client *http.Client, opts ...Option) (*http.Response, error) {
	if client == nil {
		client = http.DefaultClient
	}

	var resp *http.Response
	err := Retry(func() error {
		r, err := client.Do(req)
		if err != nil {
			return err
		}

		// Check if the status code is retryable
		if r.StatusCode >= 500 || r.StatusCode == 429 {
			_ = r.Body.Close()
			return fmt.Errorf("retryable status: %d", r.StatusCode)
		}

		resp = r
		return nil
	}, opts...)

	return resp, err
}
