package gossm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Fetch retrieves the value of key from a JSON secret stored in AWS Secrets Manager.
// arn is the full secret ARN; key is the field name within the JSON object.
//
// The package-level AWS client is used by default (initialised once from the
// standard credential chain). Pass an optional [Client] as the last argument
// to use a different region, endpoint, or credentials, or to inject a mock in
// tests:
//
//	// default client — reads credentials from env / IAM role
//	val, err := gossm.Fetch(arn, "db_password")
//
//	// custom or mock client
//	val, err := gossm.Fetch(arn, "db_password", myClient)
func Fetch(arn, key string, c ...Client) (string, error) {
	return FetchWithContext(context.Background(), arn, key, c...)
}

// FetchWithContext is like [Fetch] but propagates the caller-supplied context.
// ctx is compatible with any [context.Context] implementation — stdlib,
// Gin (c.Request.Context()), Echo (c.Request().Context()), Chi (r.Context()),
// or any other framework:
//
//	// stdlib / Chi handler
//	val, err := gossm.FetchWithContext(r.Context(), arn, "db_password")
//
//	// with a custom or mock client
//	val, err := gossm.FetchWithContext(ctx, arn, "db_password", myClient)
func FetchWithContext(ctx context.Context, arn, key string, c ...Client) (string, error) {
	if arn == "" {
		return "", ErrEmptyARN
	}
	if key == "" {
		return "", ErrEmptyKey
	}

	client, err := secretsClient(ctx, c)
	if err != nil {
		return "", fmt.Errorf("gossm: init client: %w", err)
	}

	out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(arn),
	})
	if err != nil {
		return "", &SecretError{ARN: arn, Err: err}
	}
	if out.SecretString == nil {
		return "", &SecretError{ARN: arn, Err: fmt.Errorf("secret has no string value (binary secrets are not supported)")}
	}

	val, err := extractKey(*out.SecretString, key)
	if err != nil {
		return "", &SecretError{ARN: arn, Err: err}
	}
	return val, nil
}

func extractKey(raw, key string) (string, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return "", fmt.Errorf("secret value is not valid JSON: %w", err)
	}
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("key %q value is not a string", key)
	}
	return s, nil
}
