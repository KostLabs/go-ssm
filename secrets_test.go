package gossm_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KostLabs/gossm"
)

const testARN = "arn:aws:secretsmanager:eu-west-1:123456789:secret:myapp/db"

// mockClient is an in-test Client backed by a static map.
type mockClient struct {
	secrets map[string]string
	err     error
}

func (m *mockClient) GetSecretValue(_ context.Context, input *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	id := aws.ToString(input.SecretId)
	val, ok := m.secrets[id]
	if !ok {
		return nil, fmt.Errorf("ResourceNotFoundException: %s", id)
	}
	return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(val)}, nil
}

// nilSecretClient returns a successful response with a nil SecretString.
type nilSecretClient struct{}

func (n *nilSecretClient) GetSecretValue(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return &secretsmanager.GetSecretValueOutput{SecretString: nil}, nil
}

// --- Fetch (uses context.Background internally) ---

func TestFetch_Success(t *testing.T) {
	c := &mockClient{secrets: map[string]string{
		testARN: `{"user":"pgadmin","pass":"hunter2"}`,
	}}

	user, err := gossm.Fetch(testARN, "user", c)
	require.NoError(t, err)
	assert.Equal(t, "pgadmin", user)

	pass, err := gossm.Fetch(testARN, "pass", c)
	require.NoError(t, err)
	assert.Equal(t, "hunter2", pass)
}

func TestFetch_EmptyARN(t *testing.T) {
	_, err := gossm.Fetch("", "key", &mockClient{})
	assert.ErrorIs(t, err, gossm.ErrEmptyARN)
}

func TestFetch_EmptyKey(t *testing.T) {
	_, err := gossm.Fetch(testARN, "", &mockClient{})
	assert.ErrorIs(t, err, gossm.ErrEmptyKey)
}

// --- FetchWithContext (caller-supplied context) ---

func TestFetchWithContext_Success(t *testing.T) {
	c := &mockClient{secrets: map[string]string{testARN: `{"k":"v"}`}}
	got, err := gossm.FetchWithContext(context.Background(), testARN, "k", c)
	require.NoError(t, err)
	assert.Equal(t, "v", got)
}

func TestFetchWithContext_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := &mockClient{err: ctx.Err()}
	_, err := gossm.FetchWithContext(ctx, testARN, "key", c)
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
	assert.ErrorIs(t, err, context.Canceled)
}

// --- Error paths ---

func TestFetch_ClientError(t *testing.T) {
	underlying := fmt.Errorf("network timeout")
	_, err := gossm.Fetch(testARN, "user", &mockClient{err: underlying})

	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
	assert.Equal(t, testARN, se.ARN)
	assert.ErrorIs(t, err, underlying)
}

func TestFetch_SecretNotFound(t *testing.T) {
	_, err := gossm.Fetch(testARN, "user", &mockClient{secrets: map[string]string{}})
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}

func TestFetch_NilSecretString(t *testing.T) {
	_, err := gossm.Fetch(testARN, "user", &nilSecretClient{})
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}

func TestFetch_InvalidJSON(t *testing.T) {
	c := &mockClient{secrets: map[string]string{testARN: `not-json`}}
	_, err := gossm.Fetch(testARN, "user", c)
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}

func TestFetch_KeyNotFound(t *testing.T) {
	c := &mockClient{secrets: map[string]string{testARN: `{"user":"admin"}`}}
	_, err := gossm.Fetch(testARN, "missing", c)
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}

func TestFetch_KeyNotString(t *testing.T) {
	c := &mockClient{secrets: map[string]string{testARN: `{"port":5432}`}}
	_, err := gossm.Fetch(testARN, "port", c)
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}
