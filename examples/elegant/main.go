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

// This example showcases the elegant iterator API with the new naming

func main() {
	// Example 1: Simple HTTP request with custom retry logic
	simpleHTTPRetry()
	
	// Example 2: Database migration with partial success tracking
	databaseMigration()
	
	// Example 3: Multi-region API fallback
	multiRegionFallback()
	
	// Example 4: Rate-limited API with adaptive backoff
	rateLimitedAPI()
}

func simpleHTTPRetry() {
	fmt.Println("=== Simple HTTP Retry ===")
	
	var response *http.Response
	
	for attempt := range ebo.Attempts(
		ebo.Tries(3),
		ebo.Initial(500*time.Millisecond),
	) {
		log.Printf("Attempt %d after %v delay", attempt.Number, attempt.Delay)
		
		resp, err := http.Get("https://httpbin.org/status/200")
		if err != nil {
			attempt.LastError = err
			continue
		}
		
		if resp.StatusCode == 200 {
			response = resp
			break
		}
		
		resp.Body.Close()
		attempt.LastError = fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	
	if response != nil {
		defer response.Body.Close()
		fmt.Println("Success!")
	} else {
		fmt.Println("Failed after all attempts")
	}
}

func databaseMigration() {
	fmt.Println("\n=== Database Migration ===")
	
	tables := []string{"users", "products", "orders", "invoices"}
	migrated := make(map[string]bool)
	
	for _, table := range tables {
		for attempt := range ebo.Attempts(ebo.Tries(2)) {
			log.Printf("Migrating %s (attempt %d)", table, attempt.Number)
			
			// Simulate migration with 70% success rate
			if time.Now().UnixNano()%10 < 7 {
				migrated[table] = true
				log.Printf("Successfully migrated %s", table)
				break
			}
			
			log.Printf("Failed to migrate %s, retrying...", table)
		}
	}
	
	// Report results
	successful := 0
	for table, success := range migrated {
		if success {
			successful++
		} else {
			log.Printf("Failed to migrate %s after all attempts", table)
		}
	}
	
	fmt.Printf("Migration complete: %d/%d tables migrated\n", successful, len(tables))
}

func multiRegionFallback() {
	fmt.Println("\n=== Multi-Region API Fallback ===")
	
	regions := []struct {
		name     string
		endpoint string
	}{
		{"us-east", "https://us-east.api.example.com"},
		{"eu-west", "https://eu-west.api.example.com"},
		{"ap-south", "https://ap-south.api.example.com"},
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var finalResult string
	var lastError error
	
	for _, region := range regions {
		log.Printf("Trying region: %s", region.name)
		
		success := false
		for attempt := range ebo.AttemptsWithContext(ctx,
			ebo.Tries(2),
			ebo.Initial(300*time.Millisecond),
		) {
			// Simulate API call with regional failover
			if region.name == "ap-south" && attempt.Number == 2 {
				finalResult = fmt.Sprintf("Success from %s", region.name)
				success = true
				break
			}
			
			lastError = fmt.Errorf("connection failed to %s", region.endpoint)
		}
		
		if success {
			break
		}
		
		log.Printf("Region %s failed, trying next...", region.name)
	}
	
	if finalResult != "" {
		fmt.Printf("Multi-region success: %s\n", finalResult)
	} else {
		fmt.Printf("All regions failed: %v\n", lastError)
	}
}

func rateLimitedAPI() {
	fmt.Println("\n=== Rate-Limited API ===")
	
	// Simulate API with rate limit that increases delay on 429 responses
	baseDelay := 100 * time.Millisecond
	
	err := ebo.DoWithAttempts(func(attempt *ebo.Attempt) error {
		log.Printf("API call attempt %d", attempt.Number)
		
		// Simulate API response
		if attempt.Number < 3 {
			// Rate limited
			log.Printf("Rate limited! Waiting longer...")
			time.Sleep(baseDelay * time.Duration(attempt.Number))
			return errors.New("rate limited: 429")
		}
		
		// Success
		log.Printf("API call successful")
		return nil
	}, 
		ebo.Tries(5),
		ebo.Initial(baseDelay),
		ebo.Multiplier(3.0), // Aggressive backoff for rate limits
	)
	
	if err != nil {
		fmt.Printf("API failed: %v\n", err)
	} else {
		fmt.Println("API call completed successfully")
	}
}