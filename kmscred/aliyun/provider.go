package aliyun

import "fmt"

// NewKMSClientByMode creates a new KMS client based on the mode
// mode: "aksk" or "ram"
// For "ram" mode, it uses ECS metadata service (region is optional)
// For "aksk" mode, it requires accessKey, secretKey, and region
func NewKMSClientByMode(mode, accessKey, secretKey, region string) (*KMSClient, error) {
	switch mode {
	case "ram":
		return NewKMSClient()
	case "aksk":
		if accessKey == "" || secretKey == "" {
			return nil, fmt.Errorf("accessKey and secretKey are required for aksk mode")
		}
		if region == "" {
			return nil, fmt.Errorf("region is required for aksk mode")
		}
		return NewKMSClientWithAKSK(accessKey, secretKey, region)
	default:
		return nil, fmt.Errorf("invalid mode: %s, must be 'aksk' or 'ram'", mode)
	}
}
