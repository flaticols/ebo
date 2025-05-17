package ebo

import (
	"fmt"
	"net/http"
)

// RetryMiddleware creates HTTP middleware that automatically retries requests
// based on configurable conditions. It wraps an existing http.Handler.
type RetryMiddleware struct {
	next    http.Handler
	options []Option
	checker ResponseChecker
}

// ResponseChecker is a function that determines if a response should trigger a retry
type ResponseChecker func(*http.Response) bool

// DefaultResponseChecker returns true for 5xx errors and 429 (Too Many Requests)
func DefaultResponseChecker(resp *http.Response) bool {
	return resp.StatusCode >= 500 || resp.StatusCode == 429
}

// NewRetryMiddleware creates a new retry middleware with the given options
func NewRetryMiddleware(next http.Handler, checker ResponseChecker, opts ...Option) *RetryMiddleware {
	if checker == nil {
		checker = DefaultResponseChecker
	}
	return &RetryMiddleware{
		next:    next,
		options: opts,
		checker: checker,
	}
}

// ServeHTTP implements the http.Handler interface
func (m *RetryMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create a response recorder to capture the response
	recorder := newResponseRecorder()

	err := Retry(func() error {
		// Reset the recorder for each attempt
		recorder.reset()

		// Call the next handler
		m.next.ServeHTTP(recorder, r)

		// Check if we should retry
		result := recorder.Result()
		shouldRetry := m.checker(result)
		if result.Body != nil {
			_ = result.Body.Close() // Close the body as required by bodyclose linter
		}
		if shouldRetry {
			return fmt.Errorf("retryable status: %d", recorder.Code)
		}

		return nil
	}, m.options...)

	if err != nil {
		// If all retries failed, write the last response
		recorder.writeTo(w)
		return
	}

	// Success - write the successful response
	recorder.writeTo(w)
}

// Middleware returns a middleware function compatible with popular routers
func Middleware(checker ResponseChecker, opts ...Option) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return NewRetryMiddleware(next, checker, opts...)
	}
}

// responseRecorder captures HTTP responses for retry logic
type responseRecorder struct {
	Code        int
	Headers     http.Header
	Body        []byte
	wroteHeader bool
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{
		Headers: make(http.Header),
		Code:    http.StatusOK,
	}
}

func (r *responseRecorder) reset() {
	r.Headers = make(http.Header)
	r.Body = nil
	r.Code = http.StatusOK
	r.wroteHeader = false
}

func (r *responseRecorder) Header() http.Header {
	return r.Headers
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	r.Body = append(r.Body, b...)
	return len(b), nil
}

func (r *responseRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.Code = code
	r.wroteHeader = true
}

func (r *responseRecorder) Result() *http.Response {
	return &http.Response{
		StatusCode: r.Code,
		Header:     r.Headers,
	}
}

func (r *responseRecorder) writeTo(w http.ResponseWriter) {
	// Copy headers
	for k, v := range r.Headers {
		w.Header()[k] = v
	}

	// Write status code
	w.WriteHeader(r.Code)

	// Write body
	if len(r.Body) > 0 {
		_, _ = w.Write(r.Body)
	}
}
