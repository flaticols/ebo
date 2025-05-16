package ebo

import (
	"context"
	"errors"
	"time"
)

// Attempt represents a single retry attempt
type Attempt struct {
	Number    int           // Attempt number, starting from 1
	Delay     time.Duration // Time to wait before this attempt
	Elapsed   time.Duration // Total elapsed time since first attempt
	LastError error         // Error from previous attempt (nil on first attempt)
	Context   context.Context
}

// Attempts creates an iterator that yields retry attempts with exponential backoff.
// This is ideal for building custom retry logic, implementing complex patterns,
// or when you need fine-grained control over the retry process.
//
// Example:
//
//	for attempt := range ebo.Attempts(ebo.Tries(3)) {
//	    resp, err := http.Get(url)
//	    if err == nil && resp.StatusCode == 200 {
//	        return resp, nil
//	    }
//	    log.Printf("Attempt %d failed", attempt.Number)
//	}
func Attempts(opts ...Option) func(func(*Attempt) bool) {
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

	return func(yield func(*Attempt) bool) {
		startTime := time.Now()
		attempts := 0
		var currentDelay time.Duration
		nextInterval := config.InitialInterval
		var lastError error

		for {
			attempts++
			elapsed := time.Since(startTime)

			// Check stopping conditions
			if attempts > 1 { // After first attempt
				if config.MaxRetries > 0 && attempts > config.MaxRetries {
					return
				}
				if config.MaxElapsedTime > 0 && elapsed >= config.MaxElapsedTime {
					return
				}
			}

			attempt := &Attempt{
				Number:    attempts,
				Delay:     currentDelay,
				Elapsed:   elapsed,
				LastError: lastError,
				Context:   context.Background(),
			}

			// Wait before attempt (except first one)
			if currentDelay > 0 {
				time.Sleep(currentDelay)
			}

			// Yield control to the caller
			if !yield(attempt) {
				return
			}

			// Calculate delay for next attempt
			currentDelay = nextInterval
			
			// Calculate next interval with jitter
			nextInterval = min(time.Duration(float64(nextInterval)*config.Multiplier), config.MaxInterval)
			if config.RandomizeFactor > 0 {
				delta := config.RandomizeFactor * float64(nextInterval)
				minInterval := float64(nextInterval) - delta
				maxInterval := float64(nextInterval) + delta
				nextInterval = time.Duration(minInterval + (randomFloat() * (maxInterval - minInterval)))
			}
		}
	}
}

// AttemptsWithContext creates an iterator with context support.
// The iterator will stop if the context is cancelled.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	
//	for attempt := range ebo.AttemptsWithContext(ctx) {
//	    if err := doWork(attempt.Context); err == nil {
//	        return nil
//	    }
//	}
func AttemptsWithContext(ctx context.Context, opts ...Option) func(func(*Attempt) bool) {
	baseIterator := Attempts(opts...)
	
	return func(yield func(*Attempt) bool) {
		baseIterator(func(attempt *Attempt) bool {
			// Check context cancellation
			select {
			case <-ctx.Done():
				attempt.LastError = ctx.Err()
				return false
			default:
				attempt.Context = ctx
				return yield(attempt)
			}
		})
	}
}

// DoWithAttempts provides a simple way to use the iterator pattern.
// It's a convenience wrapper around the Attempts iterator.
//
// Example:
//
//	err := ebo.DoWithAttempts(func(attempt *ebo.Attempt) error {
//	    return apiCall()
//	}, ebo.Tries(5))
func DoWithAttempts(fn func(*Attempt) error, opts ...Option) error {
	var finalErr error
	
	for attempt := range Attempts(opts...) {
		err := fn(attempt)
		if err == nil {
			return nil
		}
		
		// Check for permanent errors
		var permErr *permanentError
		if errors.As(err, &permErr) {
			return permErr.err
		}
		
		finalErr = err
		attempt.LastError = err
	}
	
	return finalErr
}

// DoWithAttemptsContext provides context-aware iteration.
// It will stop retrying if the context is cancelled.
//
// Example:
//
//	ctx := context.Background()
//	err := ebo.DoWithAttemptsContext(ctx, func(attempt *ebo.Attempt) error {
//	    return apiCall(attempt.Context)
//	}, ebo.Tries(3))
func DoWithAttemptsContext(ctx context.Context, fn func(*Attempt) error, opts ...Option) error {
	var finalErr error
	
	for attempt := range AttemptsWithContext(ctx, opts...) {
		err := fn(attempt)
		if err == nil {
			return nil
		}
		
		// Check for permanent errors
		var permErr *permanentError
		if errors.As(err, &permErr) {
			return permErr.err
		}
		
		finalErr = err
		attempt.LastError = err
	}
	
	return finalErr
}

// randomFloat returns a random float64 in [0.0, 1.0)
func randomFloat() float64 {
	return float64(time.Now().UnixNano()%1000000) / 1000000.0
}
