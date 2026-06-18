// multisecrets demonstrates fetching multiple keys from multiple AWS Secrets
// Manager ARNs into a typed Config struct. All secrets are loaded once at
// startup; the rest of the program reads from the struct.
//
// Two secrets are expected:
//
//	SECRET_ARN_DB  → {"db_user":"admin","db_password":"s3cr3t"}
//	SECRET_ARN_API → {"api_key":"abc123","api_secret":"xyz789"}
//
// Usage:
//
//	export $(cat examples/.env | xargs) && go run ./examples/multisecrets
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/KostLabs/gossm"
)

// Config holds all secrets the service needs at runtime.
type Config struct {
	DBUser     string
	DBPassword string
	APIKey     string
	APISecret  string
}

func loadConfig(ctx context.Context) (*Config, error) {
	dbARN := os.Getenv("SECRET_ARN_DB")
	apiARN := os.Getenv("SECRET_ARN_API")

	var cfg Config

	entries := []struct {
		target *string
		arn    string
		key    string
	}{
		{&cfg.DBUser, dbARN, "db_user"},
		{&cfg.DBPassword, dbARN, "db_password"},
		{&cfg.APIKey, apiARN, "api_key"},
		{&cfg.APISecret, apiARN, "api_secret"},
	}

	for _, e := range entries {
		val, err := gossm.FetchWithContext(ctx, e.arn, e.key)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.key, err)
		}
		*e.target = val
	}

	return &cfg, nil
}

func main() {
	ctx := context.Background()

	cfg, err := loadConfig(ctx)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	fmt.Printf("db_user     = %s\n", cfg.DBUser)
	fmt.Printf("db_password = %s\n", cfg.DBPassword)
	fmt.Printf("api_key     = %s\n", cfg.APIKey)
	fmt.Printf("api_secret  = %s\n", cfg.APISecret)
}
