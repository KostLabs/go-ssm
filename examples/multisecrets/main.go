// multisecrets demonstrates fetching multiple keys from multiple AWS Secrets
// Manager ARNs into a typed Config struct using FetchSecretMap. Both secrets
// are retrieved in two AWS calls — one per ARN — at startup.
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

	db, err := gossm.FetchSecretMap(ctx, dbARN)
	if err != nil {
		return nil, fmt.Errorf("db secret: %w", err)
	}
	api, err := gossm.FetchSecretMap(ctx, apiARN)
	if err != nil {
		return nil, fmt.Errorf("api secret: %w", err)
	}

	return &Config{
		DBUser:     db["db_user"],
		DBPassword: db["db_password"],
		APIKey:     api["api_key"],
		APISecret:  api["api_secret"],
	}, nil
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
