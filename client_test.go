package gossm_test

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/stretchr/testify/assert"

	"github.com/KostLabs/gossm"
)

func TestNewClientFromConfig(t *testing.T) {
	cfg := aws.Config{
		Region:      "eu-west-1",
		Credentials: credentials.NewStaticCredentialsProvider("id", "secret", ""),
	}

	c := gossm.NewClientFromConfig(cfg)

	assert.NotNil(t, c)
}
