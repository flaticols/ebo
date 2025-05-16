package ebo

import (
	"context"
	"errors"
	"iter"
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
func Attempts(opts ...Option) iter.Seq[*Attempt] {
	config := &RetryConfig{
		InitialInterval: defaultInitialInterval,
		MaxInterval:     defaultMaxInterval,
		MaxRetries:      defaultMaxRetries,
		Multiplier:      defaultMultiplier,
		MaxElapsedTime:  defaultMaxElapsedTime,
		RandomizeFactor: defaultRandomizeFactor,
	}
	
	for _, opt := range opts {
		opt(config)
	}
	
	return func(yield func(*Attempt) bool) {
		startTime := time.Now()
		currentInterval := config.InitialInterval
		elapsed := time.Duration(0)
		
		for i := 0; ; i++ {
			// Check max retries
			if config.MaxRetries > 0 && i >= config.MaxRetries {
				return
			}
			
			// Check max elapsed time
			if config.MaxElapsedTime > 0 && elapsed > config.MaxElapsedTime {
				return
			}
			
			// Create attempt with current delay
			attempt := &Attempt{
				Number:  i + 1,
				Delay:   currentInterval,
				Elapsed: elapsed,
				Context: context.Background(),
			}
			
			// For the first attempt, set delay to 0
			if i == 0 {
				attempt.Delay = 0
			}
			
			// Wait before yielding (except for first attempt)
			if i > 0 {
				time.Sleep(currentInterval)
				elapsed = time.Since(startTime)
			}
			
			// Yield attempt
			if !yield(attempt) {
				return
			}
			
			// Update interval for next iteration
			if config.Multiplier > 0 {
				currentInterval = time.Duration(float64(currentInterval) * config.Multiplier)
			}
			
			// Apply max interval cap
			if currentInterval > config.MaxInterval {
				currentInterval = config.MaxInterval
			}
			
			// Apply jitter if configured  
			if config.RandomizeFactor > 0 {
				currentInterval = getNextInterval(currentInterval, config.RandomizeFactor)
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
func AttemptsWithContext(ctx context.Context, opts ...Option) iter.Seq[*Attempt] {
	config := &RetryConfig{
		InitialInterval: defaultInitialInterval,
		MaxInterval:     defaultMaxInterval,
		MaxRetries:      defaultMaxRetries,
		Multiplier:      defaultMultiplier,
		MaxElapsedTime:  defaultMaxElapsedTime,
		RandomizeFactor: defaultRandomizeFactor,
	}
	
	for _, opt := range opts {
		opt(config)
	}
	
	return func(yield func(*Attempt) bool) {
		startTime := time.Now()
		currentInterval := config.InitialInterval
		elapsed := time.Duration(0)
		
		for i := 0; ; i++ {
			// Check context
			if ctx.Err() != nil {
				return
			}
			
			// Check max retries
			if config.MaxRetries > 0 && i >= config.MaxRetries {
				return
			}
			
			// Check max elapsed time
			if config.MaxElapsedTime > 0 && elapsed > config.MaxElapsedTime {
				return
			}
			
			// Create attempt with current delay
			attempt := &Attempt{
				Number:  i + 1,
				Delay:   currentInterval,
				Elapsed: elapsed,
				Context: ctx,
			}
			
			// For the first attempt, set delay to 0
			if i == 0 {
				attempt.Delay = 0
			}
			
			// Wait before yielding (except for first attempt)
			if i > 0 {
				select {
				case <-time.After(currentInterval):
					elapsed = time.Since(startTime)
				case <-ctx.Done():
					return
				}
			}
			
			// Yield attempt
			if !yield(attempt) {
				return
			}
			
			// Update interval for next iteration
			if config.Multiplier > 0 {
				currentInterval = time.Duration(float64(currentInterval) * config.Multiplier)
			}
			
			// Apply max interval cap
			if currentInterval > config.MaxInterval {
				currentInterval = config.MaxInterval
			}
			
			// Apply jitter if configured  
			if config.RandomizeFactor > 0 {
				currentInterval = getNextInterval(currentInterval, config.RandomizeFactor)
			}
		}
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
	var lastErr error
	
	for attempt := range Attempts(opts...) {
		if err := fn(attempt); err == nil {
			return nil
		} else {
			lastErr = err
			
			// Check if it's a permanent error
			var permanent *permanentError
			if errors.As(err, &permanent) {
				return permanent.err
			}
			attempt.LastError = err
		}
	}
	
	if lastErr != nil {
		return lastErr
	}
	return errors.New("all retry attempts failed")
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
	var lastErr error
	
	for attempt := range AttemptsWithContext(ctx, opts...) {
		if err := fn(attempt); err == nil {
			return nil
		} else {
			lastErr = err
			
			// Check if it's a permanent error
			var permanent *permanentError
			if errors.As(err, &permanent) {
				return permanent.err
			}
			attempt.LastError = err
		}
	}
	
	if ctx.Err() != nil {
		return ctx.Err()
	}
	
	if lastErr != nil {
		return lastErr
	}
	return errors.New("all retry attempts failed")
}