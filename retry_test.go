package ebo

import (
	"errors"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	t.Run("successful after retries", func(t *testing.T) {
		attempts := 0
		err := Retry(func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary error")
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected success, got error: %v", err)
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		attempts := 0
		err := Retry(func() error {
			attempts++
			return errors.New("permanent error")
		},
			WithInitialInterval(10*time.Millisecond),
			WithMaxInterval(50*time.Millisecond),
			WithMaxRetries(3),
			WithMultiplier(2.0),
		)

		if err == nil {
			t.Error("expected error, got nil")
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("with timeout", func(t *testing.T) {
		startTime := time.Now()
		err := Retry(func() error {
			return errors.New("timeout test")
		},
			WithInitialInterval(100*time.Millisecond),
			WithMaxElapsedTime(300*time.Millisecond),
		)

		elapsed := time.Since(startTime)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if elapsed > 400*time.Millisecond {
			t.Errorf("expected timeout around 300ms, took %v", elapsed)
		}
	})

	t.Run("with jitter", func(t *testing.T) {
		attempts := 0
		err := Retry(func() error {
			attempts++
			if attempts < 2 {
				return errors.New("jitter test")
			}
			return nil
		},
			WithInitialInterval(100*time.Millisecond),
			WithRandomizeFactor(0.5),
		)

		if err != nil {
			t.Errorf("expected success, got error: %v", err)
		}
	})

	t.Run("no jitter", func(t *testing.T) {
		attempts := 0
		err := Retry(func() error {
			attempts++
			if attempts < 2 {
				return errors.New("no jitter test")
			}
			return nil
		},
			WithInitialInterval(100*time.Millisecond),
			WithRandomizeFactor(0),
		)

		if err != nil {
			t.Errorf("expected success, got error: %v", err)
		}
	})

	t.Run("quick retry success", func(t *testing.T) {
		attempts := 0
		err := QuickRetry(func() error {
			attempts++
			if attempts < 2 {
				return errors.New("temporary error")
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected success, got error: %v", err)
		}
	})

	t.Run("simple backoff", func(t *testing.T) {
		attempts := 0
		err := RetryWithBackoff(func() error {
			attempts++
			if attempts < 2 {
				return errors.New("temporary error")
			}
			return nil
		}, 5)

		if err != nil {
			t.Errorf("expected success, got error: %v", err)
		}
		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}
	})
}