// variables demonstrates fetching multiple keys from one secret into separate
// Go variables. Each call extracts one field from the same JSON secret.
//
// The secret is expected to be a JSON object, e.g.:
//
//	{"db_user":"admin","db_password":"s3cr3t","api_key":"abc123"}
//
// Usage:
//
//	export $(cat examples/.env | xargs) && go run ./examples/variables
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/KostLabs/gossm"
)

func main() {
	ctx := context.Background()

	arn := os.Getenv("SECRET_ARN")
	if arn == "" {
		log.Fatal("SECRET_ARN env var is required")
	}

	dbUser, err := gossm.FetchWithContext(ctx, arn, "db_user")
	if err != nil {
		log.Fatalf("fetch db_user: %v", err)
	}

	dbPassword, err := gossm.FetchWithContext(ctx, arn, "db_password")
	if err != nil {
		log.Fatalf("fetch db_password: %v", err)
	}

	apiKey, err := gossm.FetchWithContext(ctx, arn, "api_key")
	if err != nil {
		log.Fatalf("fetch api_key: %v", err)
	}

	fmt.Printf("db_user     = %s\n", dbUser)
	fmt.Printf("db_password = %s\n", dbPassword)
	fmt.Printf("api_key     = %s\n", apiKey)
}
