package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/flaticols/ebo"
)

func main() {
	// Create an HTTP client with retry capabilities
	client := ebo.NewHTTPClient(
		ebo.Tries(3),
		ebo.Initial(1*time.Second),
		ebo.Max(10*time.Second),
	)

	// Make a request
	resp, err := client.Get("https://httpbin.org/status/500")
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Response status: %d\n", resp.StatusCode)

	// Or use HTTPDo for more control
	req, _ := http.NewRequest("GET", "https://httpbin.org/delay/1", nil)
	resp2, err := ebo.HTTPDo(req, nil,
		ebo.Tries(2),
		ebo.Initial(500*time.Millisecond),
	)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp2.Body.Close()

	fmt.Printf("Response status: %d\n", resp2.StatusCode)
}
