package ebo

import "time"

// Option is a function that configures a RetryConfig
type Option func(*RetryConfig)

// Initial sets the initial retry interval.
// This is the delay before the first retry attempt.
//
// Example:
//
//	err := ebo.Retry(fn, ebo.Initial(1*time.Second))
func Initial(d time.Duration) Option {
	return func(c *RetryConfig) {
		c.InitialInterval = d
	}
}

// Max sets the maximum retry interval.
// The delay between retries will not exceed this value.
//
// Example:
//
//	err := ebo.Retry(fn, ebo.Max(30*time.Second))
func Max(d time.Duration) Option {
	return func(c *RetryConfig) {
		c.MaxInterval = d
	}
}

// MaxTime sets the maximum total time for all retries.
// The retry process will stop after this duration, regardless of the number of attempts.
//
// Example:
//
//	err := ebo.Retry(fn, ebo.MaxTime(5*time.Minute))
func MaxTime(d time.Duration) Option {
	return func(c *RetryConfig) {
		c.MaxElapsedTime = d
	}
}

// Tries sets the maximum number of retry attempts.
// Set to 0 for unlimited retries (use with MaxTime).
//
// Example:
//
//	err := ebo.Retry(fn, ebo.Tries(3))
func Tries(n int) Option {
	return func(c *RetryConfig) {
		c.MaxRetries = n
	}
}

// Multiplier sets the backoff multiplier.
// Each retry interval is multiplied by this factor.
//
// Example:
//
//	err := ebo.Retry(fn, ebo.Multiplier(2.0)) // Double the interval each time
func Multiplier(f float64) Option {
	return func(c *RetryConfig) {
		c.Multiplier = f
	}
}

// Jitter sets the randomization factor for jitter (0-1).
// Adds randomness to prevent thundering herd problem.
//
// Example:
//
//	err := ebo.Retry(fn, ebo.Jitter(0.3)) // ±30% randomization
func Jitter(f float64) Option {
	return func(c *RetryConfig) {
		c.RandomizeFactor = f
	}
}

// NoJitter disables jitter completely.
// Useful for predictable testing or when exact timing is required.
//
// Example:
//
//	err := ebo.Retry(fn, ebo.NoJitter())
func NoJitter() Option {
	return func(c *RetryConfig) {
		c.RandomizeFactor = 0
	}
}

// Forever sets no retry limit (only time-based stopping).
// Use with MaxTime to retry continuously for a specific duration.
//
// Example:
//
//	err := ebo.Retry(fn, ebo.Forever(), ebo.MaxTime(1*time.Hour))
func Forever() Option {
	return func(c *RetryConfig) {
		c.MaxRetries = 0
	}
}

// Aggressive sets aggressive retry parameters (fast, many retries).
// Suitable for critical operations that need quick recovery.
//
// Configuration:
// - Initial: 100ms
// - Max: 5s
// - Retries: 20
// - Multiplier: 1.5
// - Jitter: 0.1
//
// Example:
//
//	err := ebo.Retry(criticalOperation, ebo.Aggressive())
func Aggressive() Option {
	return func(c *RetryConfig) {
		c.InitialInterval = 100 * time.Millisecond
		c.MaxInterval = 5 * time.Second
		c.MaxRetries = 20
		c.Multiplier = 1.5
		c.RandomizeFactor = 0.1
	}
}

// Gentle sets gentle retry parameters (slow, few retries).
// Suitable for non-critical background operations.
//
// Configuration:
// - Initial: 2s
// - Max: 30s
// - Retries: 5
// - Multiplier: 2.0
// - Jitter: 0.5
//
// Example:
//
//	err := ebo.Retry(backgroundJob, ebo.Gentle())
func Gentle() Option {
	return func(c *RetryConfig) {
		c.InitialInterval = 2 * time.Second
		c.MaxInterval = 30 * time.Second
		c.MaxRetries = 5
		c.Multiplier = 2.0
		c.RandomizeFactor = 0.5
	}
}

// Linear disables exponential backoff (constant interval).
// Each retry uses the same interval as the previous one.
//
// Example:
//
//	err := ebo.Retry(fn, ebo.Linear(), ebo.Initial(1*time.Second))
func Linear() Option {
	return func(c *RetryConfig) {
		c.Multiplier = 1.0
		c.RandomizeFactor = 0
	}
}

// Exponential sets exponential backoff with custom factor.
// The interval multiplies by this factor on each retry.
//
// Example:
//
//	err := ebo.Retry(fn, ebo.Exponential(3.0)) // Triple the interval each time
func Exponential(factor float64) Option {
	return func(c *RetryConfig) {
		c.Multiplier = factor
		c.RandomizeFactor = 0.25
	}
}

// HTTPStatus sets common HTTP retry parameters.
// Optimized for retrying HTTP requests based on status codes.
//
// Configuration:
// - Initial: 500ms
// - Max: 10s
// - Retries: 5
// - Multiplier: 2.0
// - Jitter: 0.25
//
// Example:
//
//	client := ebo.NewHTTPClient(ebo.HTTPStatus())
func HTTPStatus() Option {
	return func(c *RetryConfig) {
		c.InitialInterval = 500 * time.Millisecond
		c.MaxInterval = 10 * time.Second
		c.MaxRetries = 5
		c.Multiplier = 2.0
		c.RandomizeFactor = 0.25
	}
}

// Database sets common database retry parameters.
// Optimized for database connection and query retries.
//
// Configuration:
// - Initial: 1s
// - Max: 30s
// - Retries: 10
// - Multiplier: 2.0
// - Jitter: 0.5
// - MaxTime: 2m
//
// Example:
//
//	err := ebo.Retry(connectDB, ebo.Database())
func Database() Option {
	return func(c *RetryConfig) {
		c.InitialInterval = 1 * time.Second
		c.MaxInterval = 30 * time.Second
		c.MaxRetries = 10
		c.Multiplier = 2.0
		c.RandomizeFactor = 0.5
		c.MaxElapsedTime = 2 * time.Minute
	}
}

// API sets common API retry parameters.
// Optimized for external API calls with moderate retry strategy.
//
// Configuration:
// - Initial: 200ms
// - Max: 5s
// - Retries: 3
// - Multiplier: 2.0
// - Jitter: 0.3
//
// Example:
//
//	err := ebo.Retry(callAPI, ebo.API())
func API() Option {
	return func(c *RetryConfig) {
		c.InitialInterval = 200 * time.Millisecond
		c.MaxInterval = 5 * time.Second
		c.MaxRetries = 3
		c.Multiplier = 2.0
		c.RandomizeFactor = 0.3
	}
}

// Quick sets parameters for quick retries.
// Suitable for fast operations that should fail quickly.
//
// Configuration:
// - Initial: 50ms
// - Max: 1s
// - Retries: 3
// - Multiplier: 2.0
// - Jitter: 0.1
//
// Example:
//
//	err := ebo.Retry(checkCache, ebo.Quick())
func Quick() Option {
	return func(c *RetryConfig) {
		c.InitialInterval = 50 * time.Millisecond
		c.MaxInterval = 1 * time.Second
		c.MaxRetries = 3
		c.Multiplier = 2.0
		c.RandomizeFactor = 0.1
	}
}

// Timeout sets a timeout-based retry strategy.
// Retries indefinitely until the specified duration is reached.
//
// Example:
//
//	err := ebo.Retry(fn, ebo.Timeout(30*time.Second))
func Timeout(d time.Duration) Option {
	return func(c *RetryConfig) {
		c.MaxElapsedTime = d
		c.MaxRetries = 0 // No retry limit, only time
	}
}

