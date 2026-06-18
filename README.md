# gossm

A thin Go wrapper around **AWS Secrets Manager** that fetches individual keys from JSON secrets directly into Go variables.

## Installation

```bash
go get github.com/KostLabs/gossm
```

---

## Basic usage — one key, zero setup

gossm initialises its own AWS client from the environment automatically. No client setup required in consumer code:

```go
arn := "arn:aws:secretsmanager:eu-west-1:123456789012:secret:myapp/db"

apiKey, err := gossm.FetchSecret(ctx, arn, "api_key")
if err != nil {
    log.Fatal(err)
}
fmt.Println(apiKey)
```

---

## Multiple keys into variables

Call `FetchSecret` once per key. All keys come from the same JSON secret:

```go
dbUser, err     := gossm.FetchSecret(ctx, arn, "db_user")
dbPassword, err := gossm.FetchSecret(ctx, arn, "db_password")
apiKey, err     := gossm.FetchSecret(ctx, arn, "api_key")
```

---

## All keys at once

`FetchSecretMap` retrieves the entire JSON secret as `map[string]string` in a single AWS call:

```go
secrets, err := gossm.FetchSecretMap(ctx, arn)
if err != nil {
    log.Fatal(err)
}
fmt.Println(secrets["db_user"])
fmt.Println(secrets["api_key"])
```

---

## Multiple secrets into a typed struct

Use `FetchSecretMap` to load multiple ARNs at startup and map the results into a typed struct:

```go
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
```

---

## Using with Viper

`FetchSecretMap` pairs naturally with [Viper](https://github.com/spf13/viper). Call it once at startup and register every secret key as a Viper default — the rest of the application reads config via `viper.GetString` as usual:

```go
func loadSecrets(ctx context.Context) error {
    arn := "arn:aws:secretsmanager:eu-west-1:123456789012:secret:myapp/db"

    secrets, err := gossm.FetchSecretMap(ctx, arn)
    if err != nil {
        return err
    }
    for k, v := range secrets {
        viper.SetDefault(k, v)
    }
    return nil
}

func main() {
    viper.SetConfigName("config")
    viper.AddConfigPath(".")
    viper.ReadInConfig() // file-based config (optional)
    viper.AutomaticEnv() // env vars take precedence

    if err := loadSecrets(context.Background()); err != nil {
        log.Fatal(err)
    }

    fmt.Println(viper.GetString("db_user"))
    fmt.Println(viper.GetString("api_key"))
}
```

Secrets are registered as Viper defaults — the lowest precedence level — so a local config file or environment variable will still override them during development.

---

## Using with a web framework context

Both functions accept any `context.Context` — stdlib, Gin, Echo, Chi, etc.:

```go
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
```

---

## Custom client

Pass a `gossm.Client` as an optional last argument when you need a specific region, endpoint, or credentials:

```go
cfg, _ := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-2"))
client := gossm.NewClientFromConfig(cfg)

val, err     := gossm.FetchSecret(ctx, arn, "key", client)
secrets, err := gossm.FetchSecretMap(ctx, arn, client)
```

---

## Authentication

gossm uses the [AWS SDK v2 default credential chain](https://docs.aws.amazon.com/sdkref/latest/guide/standardized-credentials.html) — the client is wired up automatically.

### Production (recommended): IAM roles

Never ship static credentials to production. Attach an IAM role to the compute resource and the SDK picks up credentials automatically with no code changes:

| Compute | Mechanism |
|---------|-----------|
| EC2 | [Instance profile](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2_instance-profiles.html) |
| ECS / Fargate | [Task role](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-iam-roles.html) |
| Lambda | [Execution role](https://docs.aws.amazon.com/lambda/latest/dg/lambda-intro-execution-role.html) |
| EKS | [Pod Identity / IRSA](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html) |

Minimum IAM policy:

```json
{
  "Effect": "Allow",
  "Action": "secretsmanager:GetSecretValue",
  "Resource": "arn:aws:secretsmanager:REGION:ACCOUNT:secret:myapp/*"
}
```

### Local development: environment variables

For running examples or testing locally, set credentials via environment variables. See [examples/.env.example](examples/.env.example) for the full list:

```bash
cp examples/.env.example examples/.env
# edit examples/.env with your credentials and ARNs
export $(cat examples/.env | xargs)
```

---

## Error handling

```go
var se *gossm.SecretError
if errors.As(err, &se) {
    log.Printf("secret %s failed: %v", se.ARN, se.Err)
}
```

Sentinel errors for input validation:

| Error | When |
|-------|------|
| `gossm.ErrEmptyARN` | empty ARN passed to Fetch |
| `gossm.ErrEmptyKey` | empty key passed to Fetch |

---

## Testing

Implement `gossm.Client` with a lightweight mock — no AWS account needed:

```go
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

secrets, err := gossm.FetchSecretMap(ctx, arn, &mockClient{
    secrets: map[string]string{arn: `{"api_key":"test-key","db_user":"admin"}`},
})
```

---

## Examples

Runnable examples against real AWS secrets live in [examples/](examples/). See [examples/README.md](examples/README.md) for setup and usage.
