package aliyun

import (
	"fmt"
	"os"

	"github.com/aliyun/aliyun-secretsmanager-client-go/sdk"
	"github.com/aliyun/aliyun-secretsmanager-client-go/sdk/service"
)

// KMSClient wraps the Aliyun Secrets Manager client
type KMSClient struct {
	client *sdk.SecretManagerCacheClient
}

// SecretInfo represents secret information returned by KMS
type SecretInfo struct {
	SecretName  string
	SecretValue string
}

// NewKMSClient creates a new KMS client using RAM role (ECS metadata service)
func NewKMSClient() (*KMSClient, error) {
	client, err := sdk.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS client with RAM role: %w", err)
	}

	return &KMSClient{
		client: client,
	}, nil
}

// NewKMSClientWithAKSK creates a new KMS client using AccessKey and SecretKey
func NewKMSClientWithAKSK(accessKey, secretKey, region string) (*KMSClient, error) {
	client, err := sdk.NewSecretCacheClientBuilder(
		service.NewDefaultSecretManagerClientBuilder().
			Standard().
			WithAccessKey(accessKey, secretKey).
			WithRegion(region).
			Build(),
	).Build()

	if err != nil {
		return nil, fmt.Errorf("failed to create KMS client with AKSK: %w", err)
	}

	return &KMSClient{
		client: client,
	}, nil
}

// NewKMSClientWithAKSKFromEnv creates a new KMS client using AccessKey and SecretKey from environment variables
func NewKMSClientWithAKSKFromEnv(region string) (*KMSClient, error) {
	accessKey := os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID")
	secretKey := os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET")

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("ALIBABA_CLOUD_ACCESS_KEY_ID and ALIBABA_CLOUD_ACCESS_KEY_SECRET environment variables must be set")
	}

	return NewKMSClientWithAKSK(accessKey, secretKey, region)
}

// GetSecretInfo retrieves secret information by secret name
func (c *KMSClient) GetSecretInfo(secretName string) (*SecretInfo, error) {
	secretInfo, err := c.client.GetSecretInfo(secretName)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret info for %s: %w", secretName, err)
	}

	return &SecretInfo{
		SecretName:  secretName,
		SecretValue: secretInfo.SecretValue,
	}, nil
}

// GetSecretValue retrieves only the secret value by secret name
func (c *KMSClient) GetSecretValue(secretName string) (string, error) {
	secretInfo, err := c.GetSecretInfo(secretName)
	if err != nil {
		return "", err
	}
	return secretInfo.SecretValue, nil
}
