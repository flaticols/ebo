package ebo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRetryWithContext(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0

		// Cancel context after a short delay
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := RetryWithContext(ctx, func() error {
			attempts++
			return errors.New("always fail")
		}, Initial(20*time.Millisecond))

		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled error, got: %v", err)
		}

		// Should have only attempted once or twice before context was cancelled
		if attempts > 2 {
			t.Errorf("expected at most 2 attempts, got %d", attempts)
		}
	})

	t.Run("succeeds with active context", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		err := RetryWithContext(ctx, func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary error")
			}
			return nil
		}, Initial(10*time.Millisecond))

		if err != nil {
			t.Errorf("expected success, got error: %v", err)
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})
}

func TestRetryWithLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	attempts := 0

	err := RetryWithLogging(func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("attempt %d failed", attempts)
		}
		return nil
	}, logger, Initial(10*time.Millisecond))

	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "Attempt 1 failed: attempt 1 failed") {
		t.Errorf("expected log for attempt 1, got: %s", logs)
	}
	if !strings.Contains(logs, "Attempt 2 failed: attempt 2 failed") {
		t.Errorf("expected log for attempt 2, got: %s", logs)
	}
	if strings.Contains(logs, "Attempt 3 failed") {
		t.Errorf("should not log successful attempt, got: %s", logs)
	}
}

func TestRetryWithCondition(t *testing.T) {
	t.Run("permanent errors not retried", func(t *testing.T) {
		attempts := 0
		permanentErr := errors.New("permanent error")

		err := RetryWithCondition(func() error {
			attempts++
			return permanentErr
		}, func(err error) bool {
			return !errors.Is(err, permanentErr)
		}, Tries(5))

		if !errors.Is(err, permanentErr) {
			t.Errorf("expected permanent error, got: %v", err)
		}

		if attempts != 1 {
			t.Errorf("expected 1 attempt (no retry), got %d", attempts)
		}
	})

	t.Run("retryable errors are retried", func(t *testing.T) {
		attempts := 0

		err := RetryWithCondition(func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary error")
			}
			return nil
		}, func(err error) bool {
			return err != nil && err.Error() == "temporary error"
		}, Initial(10*time.Millisecond))

		if err != nil {
			t.Errorf("expected success, got error: %v", err)
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})
}

func TestHTTPRetryTransport(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))
	defer server.Close()

	client := &http.Client{
		Transport: &HTTPRetryTransport{
			Options: []Option{
				Initial(10 * time.Millisecond),
				Tries(5),
			},
		},
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "success" {
		t.Errorf("expected body 'success', got '%s'", body)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestNewHTTPClient(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(
		Initial(10*time.Millisecond),
		Tries(3),
	)

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestHTTPDo(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("done"))
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := HTTPDo(req, nil, Initial(10*time.Millisecond))
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}
