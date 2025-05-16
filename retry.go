package ebo

import (
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds the configuration for retry with exponential backoff
type RetryConfig struct {
	InitialInterval time.Duration // Initial retry interval
	MaxInterval     time.Duration // Maximum retry interval
	MaxRetries      int           // Maximum number of retry attempts (0 for no limit)
	Multiplier      float64       // Backoff multiplier (typically 2.0)
	MaxElapsedTime  time.Duration // Maximum total time for all retries (0 for no limit)
	RandomizeFactor float64       // Randomization factor for jitter (0 to 1)
}

// Option is a function that configures a RetryConfig
type Option func(*RetryConfig)

// WithInitialInterval sets the initial retry interval
func WithInitialInterval(interval time.Duration) Option {
	return func(c *RetryConfig) {
		c.InitialInterval = interval
	}
}

// WithMaxInterval sets the maximum retry interval
func WithMaxInterval(interval time.Duration) Option {
	return func(c *RetryConfig) {
		c.MaxInterval = interval
	}
}

// WithMaxRetries sets the maximum number of retry attempts
func WithMaxRetries(retries int) Option {
	return func(c *RetryConfig) {
		c.MaxRetries = retries
	}
}

// WithMultiplier sets the backoff multiplier
func WithMultiplier(multiplier float64) Option {
	return func(c *RetryConfig) {
		c.Multiplier = multiplier
	}
}

// WithMaxElapsedTime sets the maximum total time for all retries
func WithMaxElapsedTime(duration time.Duration) Option {
	return func(c *RetryConfig) {
		c.MaxElapsedTime = duration
	}
}

// WithRandomizeFactor sets the randomization factor for jitter
func WithRandomizeFactor(factor float64) Option {
	return func(c *RetryConfig) {
		c.RandomizeFactor = factor
	}
}

// RetryableFunc is a function that can be retried
type RetryableFunc func() error

// Retry executes the given function with exponential backoff
func Retry(fn RetryableFunc, opts ...Option) error {
	config := &RetryConfig{
		InitialInterval: 500 * time.Millisecond,
		MaxInterval:     30 * time.Second,
		MaxRetries:      10,
		Multiplier:      2.0,
		MaxElapsedTime:  5 * time.Minute,
		RandomizeFactor: 0.5,
	}

	for _, opt := range opts {
		opt(config)
	}

	startTime := time.Now()
	attempts := 0
	currentInterval := config.InitialInterval

	for {
		err := fn()
		if err == nil {
			return nil
		}

		attempts++

		if config.MaxRetries > 0 && attempts >= config.MaxRetries {
			return err
		}
		if config.MaxElapsedTime > 0 && time.Since(startTime) >= config.MaxElapsedTime {
			return err
		}
		nextInterval := min(time.Duration(float64(currentInterval)*config.Multiplier), config.MaxInterval)
		if config.RandomizeFactor > 0 {
			delta := config.RandomizeFactor * float64(nextInterval)
			minInterval := float64(nextInterval) - delta
			maxInterval := float64(nextInterval) + delta
			nextInterval = time.Duration(minInterval + (rand.Float64() * (maxInterval - minInterval)))
		}
		time.Sleep(currentInterval)
		currentInterval = nextInterval
	}
}

// QuickRetry is a simplified version with sensible defaults
func QuickRetry(fn RetryableFunc) error {
	return Retry(fn,
		WithInitialInterval(100*time.Millisecond),
		WithMaxInterval(5*time.Second),
		WithMaxRetries(5),
		WithMultiplier(2.0),
		WithRandomizeFactor(0.3),
	)
}

// RetryWithBackoff is a simple exponential backoff without configuration
func RetryWithBackoff(fn RetryableFunc, maxRetries int) error {
	backoff := 100 * time.Millisecond
	maxBackoff := 10 * time.Second

	for i := range maxRetries {
		if err := fn(); err == nil {
			return nil
		} else if i == maxRetries-1 {
			return err
		}

		time.Sleep(backoff)
		backoff = time.Duration(math.Min(float64(backoff*2), float64(maxBackoff)))
	}

	return nil
}
