package ebo_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/flaticols/ebo"
)

func ExampleRetry() {
	err := ebo.Retry(func() error {
		fmt.Println("Attempting operation...")
		return errors.New("temporary failure")
	}, ebo.Tries(1))
	
	if err != nil {
		fmt.Printf("Operation failed: %v\n", err)
	}
	// Output:
	// Attempting operation...
	// Operation failed: temporary failure
}

func ExampleQuickRetry() {
	attempts := 0
	err := ebo.QuickRetry(func() error {
		attempts++
		if attempts < 2 {
			return errors.New("not ready")
		}
		return nil
	})
	
	if err == nil {
		fmt.Println("Success after", attempts, "attempts")
	}
	// Output:
	// Success after 2 attempts
}

func ExampleRetryWithContext() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	err := ebo.RetryWithContext(ctx, func() error {
		fmt.Println("Trying with context...")
		return nil
	}, ebo.Initial(100*time.Millisecond))
	
	if err == nil {
		fmt.Println("Success!")
	}
	// Output:
	// Trying with context...
	// Success!
}

func ExampleNewHTTPClient() {
	// Create a client with HTTP-optimized retry settings
	client := ebo.NewHTTPClient(ebo.HTTPStatus())
	fmt.Printf("Client type: %T\n", client)
	// Output:
	// Client type: *http.Client
}

func ExampleAttempts() {
	count := 0
	for attempt := range ebo.Attempts(ebo.Tries(2)) {
		count++
		fmt.Printf("Attempt %d\n", attempt.Number)
		if count >= 2 {
			break
		}
	}
	// Output:
	// Attempt 1
	// Attempt 2
}

func ExampleInitial() {
	// Show configuration with Initial option
	err := ebo.Retry(func() error {
		return nil
	}, 
		ebo.Initial(1*time.Second),
		ebo.Tries(3),
		ebo.NoJitter(),
	)
	
	if err == nil {
		fmt.Println("Configured successfully")
	}
	// Output:
	// Configured successfully
}

func ExampleAPI() {
	// Using API preset
	err := ebo.Retry(func() error {
		fmt.Println("Using API preset")
		return nil
	}, ebo.API())
	
	if err == nil {
		fmt.Println("API preset works")
	}
	// Output:
	// Using API preset
	// API preset works
}

func ExampleRetryWithLogging() {
	logger := log.New(log.Writer(), "[RETRY] ", 0)
	
	attempts := 0
	err := ebo.RetryWithLogging(func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary")
		}
		return nil
	}, logger, ebo.Tries(3), ebo.Initial(10*time.Millisecond))
	
	if err == nil {
		fmt.Println("Success with logging")
	}
	// Output:
	// Success with logging
}