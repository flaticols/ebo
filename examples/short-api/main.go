package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/flaticols/ebo"
)

// Demonstrates the new short API for options

func main() {
	// Example 1: Simple with short API
	fmt.Println("=== Short API Examples ===")

	// Basic retry with short options
	err := ebo.Retry(func() error {
		return doWork()
	}, ebo.Tries(3), ebo.Initial(1*time.Second))

	if err != nil {
		log.Printf("Failed: %v", err)
	}

	// Using preset configurations
	fmt.Println("\n=== Preset Configurations ===")

	// Quick retry for fast operations
	err = ebo.Retry(func() error {
		return fastOperation()
	}, ebo.Quick())

	// API retry with sensible defaults
	err = ebo.Retry(func() error {
		return apiCall()
	}, ebo.API())

	// Database retry with longer timeouts
	err = ebo.Retry(func() error {
		return dbConnect()
	}, ebo.Database())

	// Aggressive retry for critical operations
	err = ebo.Retry(func() error {
		return criticalOperation()
	}, ebo.Aggressive())

	// Gentle retry for less critical operations
	err = ebo.Retry(func() error {
		return backgroundJob()
	}, ebo.Gentle())

	// Iterator with short API
	fmt.Println("\n=== Iterator with Short API ===")

	for attempt := range ebo.Attempts(
		ebo.Tries(5),
		ebo.Initial(500*time.Millisecond),
		ebo.NoJitter(),
	) {
		fmt.Printf("Attempt %d\n", attempt.Number)
		if attempt.Number >= 3 {
			break
		}
	}

	// Context with timeout
	fmt.Println("\n=== Context with Timeout ===")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = ebo.DoWithAttemptsContext(ctx, func(attempt *ebo.Attempt) error {
		return timeoutOperation()
	}, ebo.Timeout(30*time.Second))

	// HTTP with specific configuration
	fmt.Println("\n=== HTTP Retry ===")

	client := ebo.NewHTTPClient(ebo.HTTPStatus())
	fmt.Printf("Created HTTP client: %T\n", client)

	// Linear backoff (no exponential increase)
	fmt.Println("\n=== Linear Backoff ===")

	for attempt := range ebo.Attempts(
		ebo.Linear(),
		ebo.Initial(1*time.Second),
		ebo.Tries(5),
	) {
		fmt.Printf("Linear attempt %d with %v delay\n", attempt.Number, attempt.Delay)
		if attempt.Number >= 3 {
			break
		}
	}

	// Custom exponential with factor
	fmt.Println("\n=== Custom Exponential ===")

	err = ebo.Retry(func() error {
		return exponentialOperation()
	}, ebo.Exponential(3.0), ebo.Tries(4))

	// Combining options
	fmt.Println("\n=== Combined Options ===")

	err = ebo.Retry(func() error {
		return complexOperation()
	},
		ebo.Database(),             // Start with database defaults
		ebo.Tries(20),              // Override retries
		ebo.NoJitter(),             // Disable jitter
		ebo.MaxTime(5*time.Minute), // Set max time
	)

	// Forever with timeout only
	fmt.Println("\n=== Forever with Timeout ===")

	err = ebo.Retry(func() error {
		return keepTrying()
	}, ebo.Forever(), ebo.MaxTime(10*time.Minute))
}

// Mock functions for examples
func doWork() error               { return errors.New("work failed") }
func fastOperation() error        { return nil }
func apiCall() error              { return nil }
func dbConnect() error            { return nil }
func criticalOperation() error    { return nil }
func backgroundJob() error        { return nil }
func timeoutOperation() error     { return errors.New("timeout") }
func exponentialOperation() error { return nil }
func complexOperation() error     { return nil }
func keepTrying() error           { return errors.New("still trying") }
