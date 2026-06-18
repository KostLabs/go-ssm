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
	dbARN  := os.Getenv("SECRET_ARN_DB")
	apiARN := os.Getenv("SECRET_ARN_API")

	var cfg Config

	entries := []struct {
		target *string
		arn    string
		key    string
	}{
		{&cfg.DBUser,     dbARN,  "db_user"},
		{&cfg.DBPassword, dbARN,  "db_password"},
		{&cfg.APIKey,     apiARN, "api_key"},
		{&cfg.APISecret,  apiARN, "api_secret"},
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
