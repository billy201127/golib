package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// KMSClient wraps the AWS Secrets Manager client
type KMSClient struct {
	client *secretsmanager.Client
	region string
}

// SecretInfo represents secret information returned by Secrets Manager
type SecretInfo struct {
	SecretName  string
	SecretValue string
}

// NewKMSClient creates a new Secrets Manager client using IAM role (EC2 metadata service)
// It automatically uses the default credential chain which includes:
// 1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
// 2. Shared credentials file (~/.aws/credentials)
// 3. EC2 instance metadata service (IAM role)
func NewKMSClient(region string) (*KMSClient, error) {
	if region == "" {
		return nil, fmt.Errorf("region is required for RAM mode")
	}

	// Load default config which automatically uses the credential chain
	// This includes EC2 metadata service for IAM roles
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Secrets Manager client
	client := secretsmanager.NewFromConfig(cfg)

	return &KMSClient{
		client: client,
		region: region,
	}, nil
}

// NewKMSClientWithAKSK creates a new Secrets Manager client using AccessKey and SecretKey
func NewKMSClientWithAKSK(accessKey, secretKey, region string) (*KMSClient, error) {
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("accessKey and secretKey are required for AKSK mode")
	}
	if region == "" {
		return nil, fmt.Errorf("region is required for AKSK mode")
	}

	// Create credentials
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	// Load default config and override with custom credentials and region
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Secrets Manager client
	client := secretsmanager.NewFromConfig(cfg)

	return &KMSClient{
		client: client,
		region: region,
	}, nil
}

// NewKMSClientWithAKSKFromEnv creates a new Secrets Manager client using AccessKey and SecretKey from environment variables
func NewKMSClientWithAKSKFromEnv(region string) (*KMSClient, error) {
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables must be set")
	}

	return NewKMSClientWithAKSK(accessKey, secretKey, region)
}

// GetSecretInfo retrieves secret information by secret name
func (c *KMSClient) GetSecretInfo(secretName string) (*SecretInfo, error) {
	ctx := context.Background()

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := c.client.GetSecretValue(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret value for %s: %w", secretName, err)
	}

	var secretValue string
	if result.SecretString != nil {
		secretValue = *result.SecretString
	} else if result.SecretBinary != nil {
		secretValue = string(result.SecretBinary)
	}

	return &SecretInfo{
		SecretName:  secretName,
		SecretValue: secretValue,
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
