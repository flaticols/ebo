package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/flaticols/ebo"
)

// Example demonstrating advanced retry patterns
func main() {
	// Pattern 1: Custom backoff with iterator
	fmt.Println("=== Custom Backoff Pattern ===")
	customBackoff()
	
	// Pattern 2: Circuit breaker pattern
	fmt.Println("\n=== Circuit Breaker Pattern ===")
	circuitBreaker()
	
	// Pattern 3: Hedged requests (multiple concurrent attempts)
	fmt.Println("\n=== Hedged Requests Pattern ===")
	hedgedRequests()
	
	// Pattern 4: Progressive retry with fallbacks
	fmt.Println("\n=== Progressive Retry with Fallbacks ===")
	progressiveRetry()
}

func customBackoff() {
	// Create a custom backoff sequence
	backoffSequence := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
	}
	
	attempt := 0
	for a := range ebo.Attempts(ebo.Tries(len(backoffSequence))) {
		if attempt < len(backoffSequence) && attempt > 0 {
			time.Sleep(backoffSequence[attempt-1])
		}
		
		fmt.Printf("Custom backoff attempt %d\n", a.Number)
		
		if a.Number >= 3 {
			fmt.Println("Success with custom backoff!")
			break
		}
		attempt++
	}
}

type CircuitBreaker struct {
	failureThreshold int
	failures         int
	lastFailure      time.Time
	resetTimeout     time.Duration
	halfOpen         bool
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	// Check if circuit is open
	if cb.failures >= cb.failureThreshold {
		if time.Since(cb.lastFailure) < cb.resetTimeout {
			return errors.New("circuit breaker is open")
		}
		// Try half-open state
		cb.halfOpen = true
	}
	
	for attempt := range ebo.Attempts(
		ebo.Tries(3),
		ebo.Initial(100*time.Millisecond),
	) {
		err := fn()
		
		if err == nil {
			// Success - reset circuit
			cb.failures = 0
			cb.halfOpen = false
			return nil
		}
		
		// Failure
		cb.failures++
		cb.lastFailure = time.Now()
		
		if cb.halfOpen {
			// Failed in half-open state - open the circuit again
			return fmt.Errorf("circuit breaker opened after half-open failure: %w", err)
		}
		
		fmt.Printf("Circuit breaker attempt %d failed\n", attempt.Number)
	}
	
	return errors.New("all attempts failed")
}

func circuitBreaker() {
	cb := &CircuitBreaker{
		failureThreshold: 3,
		resetTimeout:     5 * time.Second,
	}
	
	// Simulate some failures
	failCount := 0
	operation := func() error {
		failCount++
		if failCount < 4 {
			return errors.New("service unavailable")
		}
		return nil
	}
	
	// Try multiple times
	for i := 0; i < 5; i++ {
		err := cb.Call(operation)
		if err != nil {
			fmt.Printf("Call %d: %v\n", i+1, err)
			time.Sleep(1 * time.Second)
		} else {
			fmt.Printf("Call %d: Success!\n", i+1)
			break
		}
	}
}

func hedgedRequests() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	results := make(chan string, 3)
	errors := make(chan error, 3)
	
	// Start multiple requests with different delays
	strategies := []struct {
		name  string
		delay time.Duration
	}{
		{"primary", 0},
		{"backup-1", 500 * time.Millisecond},
		{"backup-2", 1 * time.Second},
	}
	
	for _, strategy := range strategies {
		go func(s struct{ name string; delay time.Duration }) {
			// Wait before starting this attempt
			time.Sleep(s.delay)
			
			for attempt := range ebo.AttemptsWithContext(ctx,
				ebo.Tries(2),
				ebo.Initial(100*time.Millisecond),
			) {
				// Simulate work with 50% failure rate
				if time.Now().UnixNano()%2 == 0 {
					results <- fmt.Sprintf("%s succeeded on attempt %d", s.name, attempt.Number)
					return
				}
			}
			errors <- fmt.Errorf("%s failed all attempts", s.name)
		}(strategy)
	}
	
	// Wait for first success or all failures
	for i := 0; i < len(strategies); i++ {
		select {
		case result := <-results:
			fmt.Printf("Hedged request succeeded: %s\n", result)
			cancel() // Cancel remaining requests
			return
		case err := <-errors:
			fmt.Printf("Hedged request failed: %v\n", err)
		case <-ctx.Done():
			fmt.Println("Hedged requests timed out")
			return
		}
	}
}

func progressiveRetry() {
	endpoints := []string{
		"https://primary.example.com",
		"https://secondary.example.com", 
		"https://fallback.example.com",
	}
	
	var lastError error
	
	for i, endpoint := range endpoints {
		fmt.Printf("Trying endpoint %d: %s\n", i+1, endpoint)
		
		success := false
		for attempt := range ebo.Attempts(
			ebo.Tries(2),
			ebo.Initial(200*time.Millisecond),
		) {
			// Simulate API call
			if i == len(endpoints)-1 && attempt.Number == 2 {
				// Last endpoint succeeds on second attempt
				fmt.Printf("  Attempt %d: Success!\n", attempt.Number)
				success = true
				break
			}
			
			fmt.Printf("  Attempt %d: Failed\n", attempt.Number)
			lastError = fmt.Errorf("failed to connect to %s", endpoint)
		}
		
		if success {
			fmt.Println("Progressive retry succeeded!")
			return
		}
		
		fmt.Printf("All attempts failed for %s, trying next endpoint...\n", endpoint)
	}
	
	log.Fatalf("All endpoints failed: %v", lastError)
}