// server demonstrates the recommended production pattern: load all secrets once
// at startup into a typed Config struct, then serve them via an HTTP /info
// endpoint. The handler reads from the in-memory struct on every request —
// no repeated calls to AWS Secrets Manager.
//
// Two secrets are expected:
//
//	SECRET_ARN_DB  → {"db_user":"admin","db_password":"s3cr3t"}
//	SECRET_ARN_API → {"api_key":"abc123","api_secret":"xyz789"}
//
// Usage:
//
//	export $(cat examples/.env | xargs) && go run ./examples/server
//	curl http://localhost:8080/info
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"db_user":     cfg.DBUser,
			"db_password": cfg.DBPassword,
			"api_key":     cfg.APIKey,
			"api_secret":  cfg.APISecret,
		})
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
