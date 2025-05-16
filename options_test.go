package ebo

import (
	"errors"
	"testing"
	"time"
)

func TestShortOptions(t *testing.T) {
	t.Run("Initial", func(t *testing.T) {
		config := &RetryConfig{}
		Initial(1 * time.Second)(config)
		if config.InitialInterval != 1*time.Second {
			t.Errorf("expected 1s, got %v", config.InitialInterval)
		}
	})

	t.Run("Max", func(t *testing.T) {
		config := &RetryConfig{}
		Max(30 * time.Second)(config)
		if config.MaxInterval != 30*time.Second {
			t.Errorf("expected 30s, got %v", config.MaxInterval)
		}
	})

	t.Run("Tries", func(t *testing.T) {
		config := &RetryConfig{}
		Tries(5)(config)
		if config.MaxRetries != 5 {
			t.Errorf("expected 5, got %d", config.MaxRetries)
		}
	})

	t.Run("Jitter", func(t *testing.T) {
		config := &RetryConfig{}
		Jitter(0.3)(config)
		if config.RandomizeFactor != 0.3 {
			t.Errorf("expected 0.3, got %f", config.RandomizeFactor)
		}
	})

	t.Run("NoJitter", func(t *testing.T) {
		config := &RetryConfig{RandomizeFactor: 0.5}
		NoJitter()(config)
		if config.RandomizeFactor != 0 {
			t.Errorf("expected 0, got %f", config.RandomizeFactor)
		}
	})

	t.Run("Forever", func(t *testing.T) {
		config := &RetryConfig{MaxRetries: 10}
		Forever()(config)
		if config.MaxRetries != 0 {
			t.Errorf("expected 0, got %d", config.MaxRetries)
		}
	})

	t.Run("Linear", func(t *testing.T) {
		config := &RetryConfig{Multiplier: 2.0, RandomizeFactor: 0.5}
		Linear()(config)
		if config.Multiplier != 1.0 {
			t.Errorf("expected 1.0, got %f", config.Multiplier)
		}
		if config.RandomizeFactor != 0 {
			t.Errorf("expected 0, got %f", config.RandomizeFactor)
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		config := &RetryConfig{MaxRetries: 10}
		Timeout(5 * time.Minute)(config)
		if config.MaxElapsedTime != 5*time.Minute {
			t.Errorf("expected 5m, got %v", config.MaxElapsedTime)
		}
		if config.MaxRetries != 0 {
			t.Errorf("expected 0, got %d", config.MaxRetries)
		}
	})
}

func TestPresets(t *testing.T) {
	t.Run("Quick", func(t *testing.T) {
		config := &RetryConfig{}
		Quick()(config)
		if config.InitialInterval != 50*time.Millisecond {
			t.Errorf("expected 50ms, got %v", config.InitialInterval)
		}
		if config.MaxRetries != 3 {
			t.Errorf("expected 3, got %d", config.MaxRetries)
		}
	})

	t.Run("API", func(t *testing.T) {
		config := &RetryConfig{}
		API()(config)
		if config.InitialInterval != 200*time.Millisecond {
			t.Errorf("expected 200ms, got %v", config.InitialInterval)
		}
		if config.MaxRetries != 3 {
			t.Errorf("expected 3, got %d", config.MaxRetries)
		}
	})

	t.Run("Database", func(t *testing.T) {
		config := &RetryConfig{}
		Database()(config)
		if config.InitialInterval != 1*time.Second {
			t.Errorf("expected 1s, got %v", config.InitialInterval)
		}
		if config.MaxElapsedTime != 2*time.Minute {
			t.Errorf("expected 2m, got %v", config.MaxElapsedTime)
		}
	})

	t.Run("Aggressive", func(t *testing.T) {
		config := &RetryConfig{}
		Aggressive()(config)
		if config.MaxRetries != 20 {
			t.Errorf("expected 20, got %d", config.MaxRetries)
		}
		if config.Multiplier != 1.5 {
			t.Errorf("expected 1.5, got %f", config.Multiplier)
		}
	})

	t.Run("Gentle", func(t *testing.T) {
		config := &RetryConfig{}
		Gentle()(config)
		if config.InitialInterval != 2*time.Second {
			t.Errorf("expected 2s, got %v", config.InitialInterval)
		}
		if config.MaxRetries != 5 {
			t.Errorf("expected 5, got %d", config.MaxRetries)
		}
	})
}

func TestUsageWithRetry(t *testing.T) {
	// Test that short options work with Retry
	attempts := 0
	err := Retry(func() error {
		attempts++
		if attempts < 2 {
			return errors.New("fail")
		}
		return nil
	}, Quick())

	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestCombiningOptions(t *testing.T) {
	config := &RetryConfig{}
	
	// Apply preset then override
	Database()(config)
	Tries(50)(config)
	NoJitter()(config)
	
	if config.InitialInterval != 1*time.Second {
		t.Errorf("expected 1s from Database preset, got %v", config.InitialInterval)
	}
	if config.MaxRetries != 50 {
		t.Errorf("expected 50 from override, got %d", config.MaxRetries)
	}
	if config.RandomizeFactor != 0 {
		t.Errorf("expected 0 from NoJitter, got %f", config.RandomizeFactor)
	}
}