package ebo

import (
	"fmt"
	"errors"
	"net/http"
	"time"
)

func ExampleRetry() {
	// Example: Retry an HTTP request with custom options
	var response *http.Response
	
	err := Retry(func() error {
		resp, err := http.Get("https://api.example.com/data")
		if err != nil {
			return err
		}
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}
		response = resp
		return nil
	},
		WithInitialInterval(1*time.Second),
		WithMaxInterval(30*time.Second),
		WithMaxRetries(5),
		WithRandomizeFactor(0.5),
	)

	if err != nil {
		fmt.Printf("Failed after retries: %v\n", err)
		return
	}
	
	defer response.Body.Close()
	// Process response...
}

func ExampleRetry_withTimeout() {
	// Example: Retry with a timeout
	err := Retry(func() error {
		// Your operation here
		return errors.New("timeout example")
	},
		WithInitialInterval(500*time.Millisecond),
		WithMaxElapsedTime(5*time.Second),
		WithMaxRetries(0), // No retry limit, only time limit
	)
	
	if err != nil {
		fmt.Printf("Operation failed within timeout: %v\n", err)
	}
}

func ExampleRetry_customBackoff() {
	// Example: Custom backoff strategy
	err := Retry(func() error {
		// Your operation here
		return nil
	},
		WithInitialInterval(100*time.Millisecond),
		WithMaxInterval(10*time.Second),
		WithMultiplier(3.0), // Triple the interval each time
		WithRandomizeFactor(0), // No jitter
	)
	
	if err != nil {
		fmt.Printf("Operation failed: %v\n", err)
	}
}

func ExampleQuickRetry() {
	// Simple retry with default settings
	err := QuickRetry(func() error {
		// Your operation here
		return errors.New("temporary failure")
	})
	
	if err != nil {
		fmt.Printf("Operation failed: %v\n", err)
	}
}

func ExampleRetryWithBackoff() {
	// Simple exponential backoff
	err := RetryWithBackoff(func() error {
		// Your operation here
		return nil
	}, 3)
	
	if err != nil {
		fmt.Printf("Failed after 3 attempts: %v\n", err)
	}
}

func ExampleOption() {
	// Example: Creating reusable option sets
	fastRetryOptions := []Option{
		WithInitialInterval(50*time.Millisecond),
		WithMaxInterval(500*time.Millisecond),
		WithMaxRetries(3),
		WithMultiplier(2.0),
	}
	
	robustRetryOptions := []Option{
		WithInitialInterval(1*time.Second),
		WithMaxInterval(1*time.Minute),
		WithMaxRetries(10),
		WithMultiplier(2.0),
		WithRandomizeFactor(0.5),
		WithMaxElapsedTime(10*time.Minute),
	}
	
	// Use fast retry for quick operations
	err := Retry(func() error {
		// Quick operation
		return nil
	}, fastRetryOptions...)
	
	if err != nil {
		fmt.Printf("Fast retry failed: %v\n", err)
	}
	
	// Use robust retry for critical operations
	err = Retry(func() error {
		// Critical operation
		return nil
	}, robustRetryOptions...)
	
	if err != nil {
		fmt.Printf("Robust retry failed: %v\n", err)
	}
}