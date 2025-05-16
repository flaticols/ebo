package ebo

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAttempts(t *testing.T) {
	t.Run("basic iteration", func(t *testing.T) {
		attempts := 0
		maxAttempts := 3
		
		for attempt := range Attempts(Tries(maxAttempts), Initial(10*time.Millisecond)) {
			attempts++
			if attempt.Number != attempts {
				t.Errorf("expected attempt number %d, got %d", attempts, attempt.Number)
			}
			if attempts >= maxAttempts {
				break
			}
		}
		
		if attempts != maxAttempts {
			t.Errorf("expected %d attempts, got %d", maxAttempts, attempts)
		}
	})
	
	t.Run("delay calculation", func(t *testing.T) {
		attempts := 0
		
		for attempt := range Attempts(
			Tries(3),
			Initial(10*time.Millisecond),
			Jitter(0), // No jitter for predictable delays
		) {
			attempts++
			
			if attempts == 1 && attempt.Delay != 0 {
				t.Errorf("first attempt should have no delay, got %v", attempt.Delay)
			}
			
			if attempts == 2 && attempt.Delay != 10*time.Millisecond {
				t.Errorf("second attempt should have 10ms delay, got %v", attempt.Delay)
			}
			
			if attempts >= 3 {
				break
			}
		}
	})
	
	t.Run("early exit", func(t *testing.T) {
		attempts := 0
		
		for attempt := range Attempts(Tries(10)) {
			attempts++
			
			if attempts == 2 {
				break // Early exit
			}
			_ = attempt // Silence unused warning
		}
		
		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}
	})
}

func TestAttemptsWithContext(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0
		
		// Cancel after a short delay
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()
		
		for attempt := range AttemptsWithContext(ctx, Initial(10*time.Millisecond)) {
			attempts++
			
			if attempts > 5 {
				t.Error("too many attempts, context should have been cancelled")
				break
			}
			_ = attempt // Silence unused warning
		}
		
		if attempts < 1 || attempts > 3 {
			t.Errorf("expected 1-3 attempts, got %d", attempts)
		}
	})
	
	t.Run("context in attempt", func(t *testing.T) {
		type contextKey string
		ctx := context.WithValue(context.Background(), contextKey("key"), "value")
		
		for attempt := range AttemptsWithContext(ctx, Tries(1)) {
			value := attempt.Context.Value(contextKey("key"))
			if value != "value" {
				t.Errorf("context value not propagated")
			}
			break
		}
	})
}

func TestDoWithAttempts(t *testing.T) {
	t.Run("successful retry", func(t *testing.T) {
		attempts := 0
		
		err := DoWithAttempts(func(attempt *Attempt) error {
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
	
	t.Run("all attempts failed", func(t *testing.T) {
		attempts := 0
		
		err := DoWithAttempts(func(attempt *Attempt) error {
			attempts++
			return errors.New("persistent error")
		}, Tries(3), Initial(10*time.Millisecond))
		
		if err == nil {
			t.Error("expected error, got nil")
		}
		
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})
	
	t.Run("permanent error", func(t *testing.T) {
		attempts := 0
		permErr := errors.New("permanent error")
		
		err := DoWithAttempts(func(attempt *Attempt) error {
			attempts++
			return &permanentError{permErr}
		}, Tries(5))
		
		if err != permErr {
			t.Errorf("expected permanent error, got: %v", err)
		}
		
		if attempts != 1 {
			t.Errorf("expected 1 attempt (no retry), got %d", attempts)
		}
	})
}

func TestDoWithAttemptsContext(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0
		
		// Cancel after a short delay
		go func() {
			time.Sleep(25 * time.Millisecond)
			cancel()
		}()
		
		err := DoWithAttemptsContext(ctx, func(attempt *Attempt) error {
			attempts++
			return errors.New("always fail")
		}, Initial(10*time.Millisecond))
		
		if err == nil {
			t.Error("expected error, got nil")
		}
		
		if attempts < 1 || attempts > 4 {
			t.Errorf("expected 1-4 attempts, got %d", attempts)
		}
	})
	
	t.Run("successful with context", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0
		
		err := DoWithAttemptsContext(ctx, func(attempt *Attempt) error {
			attempts++
			if attempts < 2 {
				return errors.New("temporary error")
			}
			return nil
		}, Initial(10*time.Millisecond))
		
		if err != nil {
			t.Errorf("expected success, got error: %v", err)
		}
		
		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}
	})
}