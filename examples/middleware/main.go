package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/flaticols/ebo"
)

// simulateUnreliableAPI creates a handler that randomly fails to simulate an unreliable API
func simulateUnreliableAPI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simulate random failures (40% failure rate)
		if rand.Float32() < 0.4 {
			status := []int{500, 502, 503, 429}[rand.Intn(4)]
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  http.StatusText(status),
				"status": status,
			})
			log.Printf("Request failed with status %d", status)
			return
		}

		// Successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Success!",
			"data": map[string]interface{}{
				"id":        rand.Intn(1000),
				"timestamp": time.Now().Unix(),
			},
		})
		log.Printf("Request succeeded")
	}
}

func main() {
	// Create the base handler
	apiHandler := simulateUnreliableAPI()

	// Wrap with retry middleware using default response checker
	// This will retry on 5xx errors and 429 (Too Many Requests)
	retryHandler := ebo.NewRetryMiddleware(apiHandler, ebo.DefaultResponseChecker,
		ebo.Initial(500*time.Millisecond), // Start with 500ms delay
		ebo.Max(5*time.Second),            // Max delay of 5 seconds
		ebo.Tries(5),                      // Try up to 5 times
		ebo.Multiplier(1.5),               // Increase delay by 1.5x each time
		ebo.Jitter(0.2),                   // Add ±20% jitter to prevent thundering herd
	)

	// Create a custom response checker that also retries on 404
	customChecker := func(resp *http.Response) bool {
		return resp.StatusCode >= 500 ||
			resp.StatusCode == 429 ||
			resp.StatusCode == 404
	}

	// Alternative middleware with custom checker
	customRetryHandler := ebo.Middleware(customChecker,
		ebo.Initial(100*time.Millisecond),
		ebo.Tries(3),
		ebo.API(), // Use preset for API calls
	)(apiHandler)

	// Set up routes
	mux := http.NewServeMux()
	mux.Handle("/api/data", retryHandler)
	mux.Handle("/api/custom", customRetryHandler)

	// Add a health check endpoint without retry
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server
	log.Println("Starting server on :8080")
	log.Println("Try:")
	log.Println("  curl http://localhost:8080/api/data")
	log.Println("  curl http://localhost:8080/api/custom")
	log.Println("  curl http://localhost:8080/health")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
