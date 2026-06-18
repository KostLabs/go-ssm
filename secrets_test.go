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

// --- FetchSecret ---

func TestFetchSecret_Success(t *testing.T) {
	c := &mockClient{secrets: map[string]string{
		testARN: `{"user":"pgadmin","pass":"hunter2"}`,
	}}

	user, err := gossm.FetchSecret(context.Background(), testARN, "user", c)
	require.NoError(t, err)
	assert.Equal(t, "pgadmin", user)

	pass, err := gossm.FetchSecret(context.Background(), testARN, "pass", c)
	require.NoError(t, err)
	assert.Equal(t, "hunter2", pass)
}

func TestFetchSecret_EmptyARN(t *testing.T) {
	_, err := gossm.FetchSecret(context.Background(), "", "key", &mockClient{})
	assert.ErrorIs(t, err, gossm.ErrEmptyARN)
}

func TestFetchSecret_EmptyKey(t *testing.T) {
	_, err := gossm.FetchSecret(context.Background(), testARN, "", &mockClient{})
	assert.ErrorIs(t, err, gossm.ErrEmptyKey)
}

func TestFetchSecret_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := &mockClient{err: ctx.Err()}
	_, err := gossm.FetchSecret(ctx, testARN, "key", c)
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestFetchSecret_ClientError(t *testing.T) {
	underlying := fmt.Errorf("network timeout")
	_, err := gossm.FetchSecret(context.Background(), testARN, "user", &mockClient{err: underlying})

	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
	assert.Equal(t, testARN, se.ARN)
	assert.ErrorIs(t, err, underlying)
}

func TestFetchSecret_SecretNotFound(t *testing.T) {
	_, err := gossm.FetchSecret(context.Background(), testARN, "user", &mockClient{secrets: map[string]string{}})
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}

func TestFetchSecret_NilSecretString(t *testing.T) {
	_, err := gossm.FetchSecret(context.Background(), testARN, "user", &nilSecretClient{})
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}

func TestFetchSecret_InvalidJSON(t *testing.T) {
	c := &mockClient{secrets: map[string]string{testARN: `not-json`}}
	_, err := gossm.FetchSecret(context.Background(), testARN, "user", c)
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}

func TestFetchSecret_KeyNotFound(t *testing.T) {
	c := &mockClient{secrets: map[string]string{testARN: `{"user":"admin"}`}}
	_, err := gossm.FetchSecret(context.Background(), testARN, "missing", c)
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}

func TestFetchSecret_KeyNotString(t *testing.T) {
	c := &mockClient{secrets: map[string]string{testARN: `{"port":5432}`}}
	_, err := gossm.FetchSecret(context.Background(), testARN, "port", c)
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}

// --- FetchSecretMap ---

func TestFetchSecretMap_Success(t *testing.T) {
	c := &mockClient{secrets: map[string]string{
		testARN: `{"db_user":"admin","db_password":"s3cr3t","api_key":"abc123"}`,
	}}

	got, err := gossm.FetchSecretMap(context.Background(), testARN, c)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"db_user":     "admin",
		"db_password": "s3cr3t",
		"api_key":     "abc123",
	}, got)
}

func TestFetchSecretMap_EmptyARN(t *testing.T) {
	_, err := gossm.FetchSecretMap(context.Background(), "", &mockClient{})
	assert.ErrorIs(t, err, gossm.ErrEmptyARN)
}

func TestFetchSecretMap_NilSecretString(t *testing.T) {
	_, err := gossm.FetchSecretMap(context.Background(), testARN, &nilSecretClient{})
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}

func TestFetchSecretMap_InvalidJSON(t *testing.T) {
	c := &mockClient{secrets: map[string]string{testARN: `not-json`}}
	_, err := gossm.FetchSecretMap(context.Background(), testARN, c)
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}

func TestFetchSecretMap_NonStringValue(t *testing.T) {
	c := &mockClient{secrets: map[string]string{testARN: `{"port":5432}`}}
	_, err := gossm.FetchSecretMap(context.Background(), testARN, c)
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
}

func TestFetchSecretMap_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := &mockClient{err: ctx.Err()}
	_, err := gossm.FetchSecretMap(ctx, testARN, c)
	var se *gossm.SecretError
	require.ErrorAs(t, err, &se)
	assert.ErrorIs(t, err, context.Canceled)
}
