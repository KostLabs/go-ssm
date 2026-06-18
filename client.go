package gossm

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Client is the interface satisfied by the AWS Secrets Manager SDK client.
// Pass a custom implementation to [Fetch] or [FetchWithContext] to override
// the default — useful for testing without a real AWS account.
type Client interface {
	GetSecretValue(ctx context.Context, input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// NewClientFromConfig returns a [Client] built from an existing [aws.Config].
// Use this when you need a custom region, endpoint, or credential provider:
//
//	cfg, _ := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-2"))
//	client := gossm.NewClientFromConfig(cfg)
//	val, err := gossm.Fetch(arn, "key", client)
func NewClientFromConfig(cfg aws.Config) Client {
	return secretsmanager.NewFromConfig(cfg)
}

// defaultClient is the package-level singleton, initialised lazily on first use.
var (
	defaultClient Client
	defaultOnce   sync.Once
	defaultErr    error
)

// secretsClient returns c[0] if provided, otherwise the package-level default
// client (initialised once from the AWS credential chain).
func secretsClient(ctx context.Context, c []Client) (Client, error) {
	if len(c) > 0 {
		return c[0], nil
	}
	defaultOnce.Do(func() {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			defaultErr = err
			return
		}
		defaultClient = secretsmanager.NewFromConfig(cfg)
	})
	return defaultClient, defaultErr
}
