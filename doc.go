/*
Package gossm provides a thin wrapper around AWS Secrets Manager that fetches
individual keys from JSON secrets directly into Go variables.

The package initialises its own AWS client from the standard credential chain
(env vars, ~/.aws/credentials, IAM role, etc.) on first use — no setup
required in consumer code.

# Basic usage — fetch a single key

	arn := "arn:aws:secretsmanager:eu-west-1:123456789012:secret:myapp/db"

	password, err := gossm.FetchSecret(ctx, arn, "db_password")
	if err != nil {
		log.Fatal(err)
	}

# Multiple keys into variables

Call [FetchSecret] once per key. Each call reads the same JSON secret:

	dbUser, err := gossm.FetchSecret(ctx, arn, "db_user")
	dbPass, err := gossm.FetchSecret(ctx, arn, "db_password")
	apiKey, err := gossm.FetchSecret(ctx, arn, "api_key")

# All keys at once

Use [FetchSecretMap] to retrieve the entire JSON secret as map[string]string in
a single AWS call:

	secrets, err := gossm.FetchSecretMap(ctx, arn)
	if err != nil {
		log.Fatal(err)
	}
	// secrets["db_user"], secrets["db_password"], …

This integrates cleanly with Viper:

	secrets, err := gossm.FetchSecretMap(ctx, arn)
	if err != nil {
		log.Fatal(err)
	}
	for k, v := range secrets {
		viper.SetDefault(k, v)
	}

# Multiple secrets into a typed struct

Declare a Config struct and populate it once at startup:

	type Config struct {
		DBUser     string
		DBPassword string
		APIKey     string
		APISecret  string
	}

	func loadConfig(ctx context.Context) (*Config, error) {
		dbARN  := "arn:aws:secretsmanager:eu-west-1:123456789012:secret:myapp/db"
		apiARN := "arn:aws:secretsmanager:eu-west-1:123456789012:secret:myapp/api"

		db, err := gossm.FetchSecretMap(ctx, dbARN)
		if err != nil {
			return nil, err
		}
		api, err := gossm.FetchSecretMap(ctx, apiARN)
		if err != nil {
			return nil, err
		}

		return &Config{
			DBUser:     db["db_user"],
			DBPassword: db["db_password"],
			APIKey:     api["api_key"],
			APISecret:  api["api_secret"],
		}, nil
	}

# Using with a web framework context

[FetchSecret] accepts any [context.Context], integrating transparently with
Gin, Echo, Chi, or stdlib handlers:

	// Gin
	func handler(c *gin.Context) {
		val, err := gossm.FetchSecret(c.Request.Context(), arn, "key")
	}

	// Echo
	func handler(c echo.Context) error {
		val, err := gossm.FetchSecret(c.Request().Context(), arn, "key")
	}

	// Chi / stdlib
	func handler(w http.ResponseWriter, r *http.Request) {
		val, err := gossm.FetchSecret(r.Context(), arn, "key")
	}

# Custom client

Pass a [Client] as an optional last argument to override the default client —
useful for a different region, a custom endpoint, or a specific credential
provider:

	cfg, _ := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-2"))
	client := gossm.NewClientFromConfig(cfg)

	val, err := gossm.FetchSecret(ctx, arn, "key", client)
	secrets, err := gossm.FetchSecretMap(ctx, arn, client)

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

	val, err := gossm.FetchSecret(ctx, arn, "api_key", &mockClient{
		secrets: map[string]string{arn: `{"api_key":"test-key"}`},
	})
*/
package gossm
