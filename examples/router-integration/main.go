package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/flaticols/ebo"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-pkgz/routegroup"
)

// API handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func unstableHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate 30% failure rate
	if rand.Float32() < 0.3 {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "service temporarily unavailable"})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Success",
		"data":    rand.Intn(1000),
		"time":    time.Now().Format(time.RFC3339),
	})
}

func rateLimitedHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate rate limiting - 50% of requests
	if rand.Float32() < 0.5 {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Request successful"})
}

// Example 1: Using with chi router
func setupChiRouter() http.Handler {
	r := chi.NewRouter()
	
	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	
	// Create EBO retry middleware
	retryMiddleware := ebo.Middleware(ebo.DefaultResponseChecker,
		ebo.Initial(200*time.Millisecond),
		ebo.Tries(3),
		ebo.Jitter(0.3),
	)
	
	// Routes without retry
	r.Get("/health", healthHandler)
	
	// Routes with retry middleware
	r.Group(func(r chi.Router) {
		r.Use(retryMiddleware) // Apply retry to this group
		
		r.Get("/api/unstable", unstableHandler)
		r.Get("/api/rate-limited", rateLimitedHandler)
		
		// Custom retry for specific endpoint
		r.With(ebo.Middleware(func(resp *http.Response) bool {
			// Only retry on 503 for this endpoint
			return resp.StatusCode == http.StatusServiceUnavailable
		}, ebo.Quick())).Get("/api/custom", unstableHandler)
	})
	
	return r
}

// Example 2: Using with routegroup
func setupRouteGroup() http.Handler {
	// Create base router
	router := routegroup.New(http.NewServeMux())
	
	// Create EBO retry middleware
	retryMiddleware := ebo.Middleware(ebo.DefaultResponseChecker,
		ebo.Initial(100*time.Millisecond),
		ebo.Max(2*time.Second),
		ebo.Tries(5),
		ebo.Multiplier(1.5),
	)
	
	// Mount without retry
	router.Mount("/health", healthHandler)
	
	// Create API group with retry middleware
	apiGroup := router.Group()
	apiGroup.Use(retryMiddleware)
	
	// Add routes to the group
	apiGroup.HandleFunc("GET /api/unstable", unstableHandler)
	apiGroup.HandleFunc("GET /api/rate-limited", rateLimitedHandler)
	
	// Create a sub-group with additional middleware
	v2Group := apiGroup.Group()
	v2Group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-API-Version", "v2")
			next.ServeHTTP(w, r)
		})
	})
	v2Group.HandleFunc("GET /api/v2/data", unstableHandler)
	
	return router
}

// Example 3: Advanced retry configuration with multiple middleware
func setupAdvancedRouter() http.Handler {
	r := chi.NewRouter()
	
	// Different retry configurations for different scenarios
	
	// Fast retry for user-facing endpoints
	fastRetry := ebo.Middleware(ebo.DefaultResponseChecker,
		ebo.Initial(50*time.Millisecond),
		ebo.Tries(2),
		ebo.Max(200*time.Millisecond),
	)
	
	// Aggressive retry for background jobs
	aggressiveRetry := ebo.Middleware(func(resp *http.Response) bool {
		// Retry on any error except 4xx client errors
		return resp.StatusCode >= 500 || resp.StatusCode == 429
	},
		ebo.Initial(1*time.Second),
		ebo.Tries(10),
		ebo.Max(30*time.Second),
		ebo.Jitter(0.5),
	)
	
	// Custom retry for external API calls
	externalAPIRetry := ebo.Middleware(func(resp *http.Response) bool {
		// Also retry on 502 Bad Gateway and 504 Gateway Timeout
		return resp.StatusCode >= 500 || 
			resp.StatusCode == 429 || 
			resp.StatusCode == 502 || 
			resp.StatusCode == 504
	},
		ebo.API(), // Use API preset
	)
	
	// Apply different retry strategies
	r.Group(func(r chi.Router) {
		r.Use(fastRetry)
		r.Get("/api/user/{id}", unstableHandler)
		r.Post("/api/user", unstableHandler)
	})
	
	r.Group(func(r chi.Router) {
		r.Use(aggressiveRetry)
		r.Post("/api/jobs/process", unstableHandler)
		r.Post("/api/jobs/batch", unstableHandler)
	})
	
	r.Group(func(r chi.Router) {
		r.Use(externalAPIRetry)
		r.Get("/api/external/weather", unstableHandler)
		r.Get("/api/external/stocks", unstableHandler)
	})
	
	return r
}

func main() {
	// Example 1: Chi router
	log.Println("Starting Chi router example on :8080")
	go func() {
		if err := http.ListenAndServe(":8080", setupChiRouter()); err != nil {
			log.Fatal(err)
		}
	}()
	
	// Example 2: RouteGroup
	log.Println("Starting RouteGroup example on :8081")
	go func() {
		if err := http.ListenAndServe(":8081", setupRouteGroup()); err != nil {
			log.Fatal(err)
		}
	}()
	
	// Example 3: Advanced configuration
	log.Println("Starting Advanced router example on :8082")
	if err := http.ListenAndServe(":8082", setupAdvancedRouter()); err != nil {
		log.Fatal(err)
	}
}