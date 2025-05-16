package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/flaticols/ebo"
)

func main() {
	// Example 1: Simple retry with iterator
	fmt.Println("Example 1: Simple retry with iterator")
	for attempt := range ebo.Attempts(
		ebo.Tries(3),
		ebo.Initial(1*time.Second),
	) {
		fmt.Printf("Attempt %d (after %v delay)\n", attempt.Number, attempt.Delay)
		
		// Simulate work
		if attempt.Number < 3 {
			attempt.LastError = errors.New("temporary failure")
			continue
		}
		
		fmt.Println("Success!")
		break
	}
	
	fmt.Println("\nExample 2: HTTP request with custom retry logic")
	var response *http.Response
	for attempt := range ebo.Attempts(
		ebo.Tries(5),
		ebo.Initial(500*time.Millisecond),
	) {
		fmt.Printf("HTTP attempt %d\n", attempt.Number)
		
		resp, err := http.Get("https://httpbin.org/status/503")
		if err != nil {
			attempt.LastError = err
			continue
		}
		
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			attempt.LastError = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}
		
		response = resp
		break
	}
	
	if response != nil {
		defer response.Body.Close()
		fmt.Printf("HTTP Success! Status: %d\n", response.StatusCode)
	} else {
		fmt.Println("HTTP request failed after all retries")
	}
	
	// Example 3: Context-aware retry
	fmt.Println("\nExample 3: Context-aware retry")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	for attempt := range ebo.AttemptsWithContext(ctx,
		ebo.Initial(500*time.Millisecond),
		ebo.Tries(10),
	) {
		fmt.Printf("Context attempt %d (elapsed: %v)\n", attempt.Number, attempt.Elapsed)
		
		// Simulate slow operation
		select {
		case <-ctx.Done():
			fmt.Println("Context cancelled!")
			return // Exit the loop on context cancellation
		case <-time.After(200 * time.Millisecond):
			// Continue trying
		}
		
		if attempt.Number == 4 {
			fmt.Println("Success before timeout!")
			break
		}
	}
	
	// Example 4: Using helper function
	fmt.Println("\nExample 4: Using DoWithAttempts helper")
	err := ebo.DoWithAttempts(func(attempt *ebo.Attempt) error {
		fmt.Printf("Helper attempt %d\n", attempt.Number)
		
		if attempt.Number < 3 {
			return errors.New("not ready yet")
		}
		
		return nil
	}, ebo.Tries(5), ebo.Initial(300*time.Millisecond))
	
	if err != nil {
		log.Printf("Failed: %v", err)
	} else {
		fmt.Println("Helper succeeded!")
	}
	
	// Example 5: Complex retry with state
	fmt.Println("\nExample 5: Stateful retry")
	var totalAttempts int
	var successCount int
	
	for attempt := range ebo.Attempts(ebo.Tries(5)) {
		totalAttempts++
		
		// Simulate work with 60% success rate
		if time.Now().UnixNano()%10 < 6 {
			successCount++
			if successCount >= 2 {
				fmt.Printf("Success after %d attempts (2 successful)\n", totalAttempts)
				break
			}
		}
		
		fmt.Printf("Attempt %d: %d successful so far\n", attempt.Number, successCount)
	}
}