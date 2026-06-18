// basic demonstrates the simplest gossm usage: fetch a single key from one
// AWS Secrets Manager secret and print it. No client setup required — gossm
// initialises one from the environment automatically.
//
// The secret is expected to be a JSON object, e.g.:
//
//	{"api_key":"abc123"}
//
// Usage:
//
//	export $(cat examples/.env | xargs) && go run ./examples/basic
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/KostLabs/gossm"
)

func main() {
	arn := os.Getenv("SECRET_ARN")
	if arn == "" {
		log.Fatal("SECRET_ARN env var is required")
	}

	apiKey, err := gossm.FetchSecret(context.Background(), arn, "api_key")
	if err != nil {
		log.Fatalf("fetch api_key: %v", err)
	}

	fmt.Printf("api_key = %s\n", apiKey)
}
