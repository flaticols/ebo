package ebo

import (
	"errors"
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


// RetryableFunc is a function that can be retried
type RetryableFunc func() error

// Retry executes the given function with exponential backoff.
// It will retry the function until it succeeds, reaches the maximum retry limit,
// or the maximum elapsed time is exceeded.
//
// Example:
//
//	err := ebo.Retry(func() error {
//	    resp, err := http.Get("https://api.example.com/data")
//	    if err != nil {
//	        return err
//	    }
//	    if resp.StatusCode >= 500 {
//	        return fmt.Errorf("server error: %d", resp.StatusCode)
//	    }
//	    return nil
//	}, ebo.Tries(5), ebo.Initial(1*time.Second))
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

		// Check if the error is permanent and should not be retried
		var permErr *permanentError
		if errors.As(err, &permErr) {
			return permErr.err
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

// QuickRetry is a simplified version with sensible defaults for quick operations.
// It uses shorter intervals and fewer retries suitable for fast operations.
//
// Default configuration:
// - Initial interval: 100ms
// - Max interval: 5s
// - Max retries: 5
// - Multiplier: 2.0
// - Jitter: 0.3
//
// Example:
//
//	err := ebo.QuickRetry(func() error {
//	    return checkServiceHealth()
//	})
func QuickRetry(fn RetryableFunc) error {
	return Retry(fn,
		Initial(100*time.Millisecond),
		Max(5*time.Second),
		Tries(5),
		Multiplier(2.0),
		Jitter(0.3),
	)
}

// RetryWithBackoff is a simple exponential backoff without configuration.
// It provides a basic retry mechanism with fixed exponential backoff.
//
// Parameters:
// - Initial interval: 100ms
// - Max interval: 10s
// - Multiplier: 2.0
//
// Example:
//
//	err := ebo.RetryWithBackoff(func() error {
//	    return performOperation()
//	}, 3) // max 3 retries
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
