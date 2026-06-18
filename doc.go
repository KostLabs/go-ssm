/*
Package gossm provides a thin wrapper around AWS Secrets Manager that fetches
individual keys from JSON secrets directly into Go variables.

The package initialises its own AWS client from the standard credential chain
(env vars, ~/.aws/credentials, IAM role, etc.) on first use — no setup
required in consumer code.

# Basic usage — fetch a single key

	arn := "arn:aws:secretsmanager:eu-west-1:123456789012:secret:myapp/db"

	password, err := gossm.Fetch(arn, "db_password")
	if err != nil {
		log.Fatal(err)
	}

# Multiple keys into variables

Call [Fetch] once per key. Each call reads the same JSON secret:

	dbUser, err := gossm.Fetch(arn, "db_user")
	dbPass, err := gossm.Fetch(arn, "db_password")
	apiKey, err := gossm.Fetch(arn, "api_key")

# Multiple secrets into a typed struct

Declare a Config struct and populate it once at startup. The rest of the
program reads from the struct without hitting AWS again:

	type Config struct {
		DBUser     string
		DBPassword string
		APIKey     string
		APISecret  string
	}

	func loadConfig(ctx context.Context) (*Config, error) {
		dbARN  := "arn:aws:secretsmanager:eu-west-1:123456789012:secret:myapp/db"
		apiARN := "arn:aws:secretsmanager:eu-west-1:123456789012:secret:myapp/api"

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

# Using with a web framework context

[FetchWithContext] accepts any [context.Context], integrating transparently
with Gin, Echo, Chi, or stdlib handlers:

	// Gin
	func handler(c *gin.Context) {
		val, err := gossm.FetchWithContext(c.Request.Context(), arn, "key")
	}

	// Echo
	func handler(c echo.Context) error {
		val, err := gossm.FetchWithContext(c.Request().Context(), arn, "key")
	}

	// Chi / stdlib
	func handler(w http.ResponseWriter, r *http.Request) {
		val, err := gossm.FetchWithContext(r.Context(), arn, "key")
	}

# Custom client

Pass a [Client] as an optional last argument to override the default client —
useful for a different region, a custom endpoint, or a specific credential
provider:

	cfg, _ := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-2"))
	client := gossm.NewClientFromConfig(cfg)

	val, err := gossm.Fetch(arn, "key", client)
	val, err := gossm.FetchWithContext(ctx, arn, "key", client)

# Error handling

All errors are typed and unwrappable:

	var se *gossm.SecretError
	if errors.As(err, &se) {
		log.Printf("secret %s failed: %v", se.ARN, se.Err)
	}

Sentinel input-validation errors: [ErrEmptyARN], [ErrEmptyKey].

# Testing

Satisfy the [Client] interface with a lightweight in-process mock — no AWS
account or credentials needed:

	type mockClient struct{ secrets map[string]string }

	func (m *mockClient) GetSecretValue(_ context.Context, input *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
		val, ok := m.secrets[aws.ToString(input.SecretId)]
		if !ok {
			return nil, fmt.Errorf("secret not found")
		}
		return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(val)}, nil
	}

	val, err := gossm.Fetch(arn, "api_key", &mockClient{
		secrets: map[string]string{arn: `{"api_key":"test-key"}`},
	})
*/
package gossm
