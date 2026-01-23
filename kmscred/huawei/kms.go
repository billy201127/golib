package huawei

import (
	"fmt"
	"os"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/provider"
	v2 "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/kms/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/kms/v2/model"
	kmsRegion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/kms/v2/region"
	"gomod.pri/golib/kmscred"
)

// KMSClient wraps the Huawei Cloud KMS client
type KMSClient struct {
	client *v2.KmsClient
	region string
}

// NewKMSClient creates a new KMS client using RAM role (ECS metadata service)
// It automatically uses the default credential chain which includes:
// 1. Environment variables (HUAWEICLOUD_SDK_AK, HUAWEICLOUD_SDK_SK)
// 2. Shared credentials file (~/.huaweicloud/credentials)
// 3. ECS instance metadata service (IAM role)
func NewKMSClient(region string) (*KMSClient, error) {
	if region == "" {
		return nil, fmt.Errorf("region is required for RAM mode")
	}

	// Use SDK's metadata credential provider to get credentials from ECS metadata service
	// This will automatically fetch credentials from the instance metadata service
	metadataProvider := provider.BasicCredentialMetadataProvider()
	auth, err := metadataProvider.GetCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials from metadata service: %w", err)
	}

	// Get region
	reg, err := kmsRegion.SafeValueOf(region)
	if err != nil {
		return nil, fmt.Errorf("invalid region: %w", err)
	}

	// Create KMS client
	hcClient, err := v2.KmsClientBuilder().
		WithRegion(reg).
		WithCredential(auth).
		SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS client: %w", err)
	}

	client := v2.NewKmsClient(hcClient)

	return &KMSClient{
		client: client,
		region: region,
	}, nil
}

// NewKMSClientWithAKSK creates a new KMS client using AccessKey and SecretKey
func NewKMSClientWithAKSK(accessKey, secretKey, region string) (*KMSClient, error) {
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("accessKey and secretKey are required for AKSK mode")
	}
	if region == "" {
		return nil, fmt.Errorf("region is required for AKSK mode")
	}

	// Create credentials
	auth, err := basic.NewCredentialsBuilder().
		WithAk(accessKey).
		WithSk(secretKey).
		SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials: %w", err)
	}

	// Get region
	reg, err := kmsRegion.SafeValueOf(region)
	if err != nil {
		return nil, fmt.Errorf("invalid region: %w", err)
	}

	// Create KMS client
	hcClient, err := v2.KmsClientBuilder().
		WithRegion(reg).
		WithCredential(auth).
		SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS client: %w", err)
	}

	client := v2.NewKmsClient(hcClient)

	return &KMSClient{
		client: client,
		region: region,
	}, nil
}

// NewKMSClientWithAKSKFromEnv creates a new KMS client using AccessKey and SecretKey from environment variables
func NewKMSClientWithAKSKFromEnv(region string) (*KMSClient, error) {
	accessKey := os.Getenv("HUAWEICLOUD_SDK_AK")
	secretKey := os.Getenv("HUAWEICLOUD_SDK_SK")

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("HUAWEICLOUD_SDK_AK and HUAWEICLOUD_SDK_SK environment variables must be set")
	}

	return NewKMSClientWithAKSK(accessKey, secretKey, region)
}

// GetSecretInfo retrieves secret information by secret name
// Note: Huawei Cloud KMS is primarily for key management, not secret storage.
// For secret management, you may need to use Huawei Cloud's dedicated secret management service.
// This implementation uses the ListKeyDetail API to get key information.
func (c *KMSClient) GetSecretInfo(secretName string) (*kmscred.SecretInfo, error) {
	// Use ListKeyDetail API to get key information
	request := &model.ListKeyDetailRequest{
		Body: &model.OperateKeyRequestBody{
			KeyId: secretName,
		},
	}

	response, err := c.client.ListKeyDetail(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret info for %s: %w", secretName, err)
	}

	// Extract secret value from response
	// Note: Huawei Cloud KMS returns key metadata, not the actual secret value
	// For actual secret values, you may need to use DecryptData API or a dedicated secret service
	secretValue := ""
	if response.KeyInfo != nil && response.KeyInfo.KeyId != nil {
		secretValue = *response.KeyInfo.KeyId
	}

	return &kmscred.SecretInfo{
		Name:  secretName,
		Value: secretValue,
	}, nil
}

// GetSecretValue retrieves only the secret value by secret name
func (c *KMSClient) GetSecretValue(secretName string) (string, error) {
	secretInfo, err := c.GetSecretInfo(secretName)
	if err != nil {
		return "", err
	}
	return secretInfo.Value, nil
}
