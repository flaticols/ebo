package ebo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryMiddleware(t *testing.T) {
	t.Run("successful request no retry", func(t *testing.T) {
		attempts := int32(0)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})

		middleware := NewRetryMiddleware(handler, DefaultResponseChecker, Initial(50*time.Millisecond))

		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if atomic.LoadInt32(&attempts) != 1 {
			t.Errorf("expected 1 attempt, got %d", atomic.LoadInt32(&attempts))
		}
		if body := rec.Body.String(); body != "success" {
			t.Errorf("expected body 'success', got %s", body)
		}
	})

	t.Run("retry on 500 error", func(t *testing.T) {
		attempts := int32(0)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			current := atomic.AddInt32(&attempts, 1)
			if current < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error"))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			}
		})

		middleware := NewRetryMiddleware(handler, DefaultResponseChecker,
			Initial(50*time.Millisecond),
			Tries(5))

		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if atomic.LoadInt32(&attempts) != 3 {
			t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
		}
		if body := rec.Body.String(); body != "success" {
			t.Errorf("expected body 'success', got %s", body)
		}
	})

	t.Run("retry on 429 too many requests", func(t *testing.T) {
		attempts := int32(0)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			current := atomic.AddInt32(&attempts, 1)
			if current < 2 {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("rate limited"))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			}
		})

		middleware := NewRetryMiddleware(handler, DefaultResponseChecker,
			Initial(50*time.Millisecond),
			Tries(3))

		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if atomic.LoadInt32(&attempts) != 2 {
			t.Errorf("expected 2 attempts, got %d", atomic.LoadInt32(&attempts))
		}
	})

	t.Run("custom response checker", func(t *testing.T) {
		attempts := int32(0)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			current := atomic.AddInt32(&attempts, 1)
			if current < 2 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("bad request"))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			}
		})

		// Custom checker that retries on 400 errors
		customChecker := func(resp *http.Response) bool {
			return resp.StatusCode == http.StatusBadRequest
		}

		middleware := NewRetryMiddleware(handler, customChecker,
			Initial(50*time.Millisecond),
			Tries(3))

		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if atomic.LoadInt32(&attempts) != 2 {
			t.Errorf("expected 2 attempts, got %d", atomic.LoadInt32(&attempts))
		}
	})

	t.Run("headers preserved", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Header", "test-value")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})

		middleware := NewRetryMiddleware(handler, DefaultResponseChecker)

		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Header().Get("X-Custom-Header") != "test-value" {
			t.Errorf("expected custom header to be preserved")
		}
		if rec.Header().Get("Content-Type") != "application/json" {
			t.Errorf("expected content-type to be preserved")
		}
	})

	t.Run("middleware function", func(t *testing.T) {
		attempts := int32(0)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			current := atomic.AddInt32(&attempts, 1)
			if current < 2 {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		})

		// Test the middleware function
		middleware := Middleware(DefaultResponseChecker,
			Initial(50*time.Millisecond),
			Tries(3))

		wrappedHandler := middleware(handler)

		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if atomic.LoadInt32(&attempts) != 2 {
			t.Errorf("expected 2 attempts, got %d", atomic.LoadInt32(&attempts))
		}
	})
}

func TestResponseRecorder(t *testing.T) {
	t.Run("basic recording", func(t *testing.T) {
		recorder := newResponseRecorder()

		recorder.Header().Set("Content-Type", "text/plain")
		recorder.WriteHeader(http.StatusCreated)
		recorder.Write([]byte("test body"))

		if recorder.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", recorder.Code)
		}
		if string(recorder.Body) != "test body" {
			t.Errorf("expected body 'test body', got %s", string(recorder.Body))
		}
		if recorder.Header().Get("Content-Type") != "text/plain" {
			t.Errorf("expected content-type header")
		}
	})

	t.Run("default status code", func(t *testing.T) {
		recorder := newResponseRecorder()
		recorder.Write([]byte("test"))

		if recorder.Code != http.StatusOK {
			t.Errorf("expected default status 200, got %d", recorder.Code)
		}
	})

	t.Run("reset functionality", func(t *testing.T) {
		recorder := newResponseRecorder()

		recorder.Header().Set("Test", "value")
		recorder.WriteHeader(http.StatusNotFound)
		recorder.Write([]byte("error"))

		recorder.reset()

		if recorder.Code != http.StatusOK {
			t.Errorf("expected status reset to 200, got %d", recorder.Code)
		}
		if len(recorder.Body) != 0 {
			t.Errorf("expected body to be reset")
		}
		if recorder.Header().Get("Test") != "" {
			t.Errorf("expected headers to be reset")
		}
	})
}

func BenchmarkRetryMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	middleware := NewRetryMiddleware(handler, DefaultResponseChecker,
		Initial(10*time.Millisecond),
		Tries(3))

	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
	}
}

func ExampleMiddleware() {
	// Create a standard HTTP handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate occasional failures
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	})

	// Wrap with retry middleware
	retryHandler := Middleware(DefaultResponseChecker,
		Initial(100*time.Millisecond),
		Tries(3),
		Jitter(0.3),
	)(handler)

	// Use with http.ListenAndServe
	// http.ListenAndServe(":8080", retryHandler)
	_ = retryHandler
}

func ExampleRetryMiddleware() {
	// Create a handler that fails sometimes
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "Hello, World!")
	})

	// Create retry middleware with custom settings
	middleware := NewRetryMiddleware(handler, DefaultResponseChecker,
		Initial(500*time.Millisecond),
		Max(5*time.Second),
		Tries(5),
		Multiplier(1.5),
	)

	// Create a test server
	server := httptest.NewServer(middleware)
	defer server.Close()

	// Make a request
	resp, err := http.Get(server.URL)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	println(string(body))
}
