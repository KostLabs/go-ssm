package gossm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// FetchSecret retrieves the value of key from a JSON secret stored in AWS Secrets Manager.
// arn is the full secret ARN; key is the field name within the JSON object.
//
// The package-level AWS client is used by default (initialised once from the
// standard credential chain). Pass an optional [Client] as the last argument
// to use a different region, endpoint, or credentials, or to inject a mock in
// tests:
//
//	val, err := gossm.FetchSecret(ctx, arn, "db_password")
//	val, err := gossm.FetchSecret(ctx, arn, "db_password", myClient)
func FetchSecret(ctx context.Context, arn, key string, c ...Client) (string, error) {
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

// FetchSecretMap fetches all string keys from a JSON secret and returns them as
// a map. A single AWS call retrieves the whole secret — no per-key round trips.
//
//	secrets, err := gossm.FetchSecretMap(ctx, arn)
//	// secrets["db_user"], secrets["db_password"], …
func FetchSecretMap(ctx context.Context, arn string, c ...Client) (map[string]string, error) {
	if arn == "" {
		return nil, ErrEmptyARN
	}

	client, err := secretsClient(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("gossm: init client: %w", err)
	}

	out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(arn),
	})
	if err != nil {
		return nil, &SecretError{ARN: arn, Err: err}
	}
	if out.SecretString == nil {
		return nil, &SecretError{ARN: arn, Err: fmt.Errorf("secret has no string value (binary secrets are not supported)")}
	}

	result, err := extractAllKeys(*out.SecretString)
	if err != nil {
		return nil, &SecretError{ARN: arn, Err: err}
	}
	return result, nil
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

func extractAllKeys(raw string) (map[string]string, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("secret value is not valid JSON: %w", err)
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("key %q value is not a string", k)
		}
		result[k] = s
	}
	return result, nil
}
