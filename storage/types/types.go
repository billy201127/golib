package types

type StorageProvider string

const (
	StorageProviderOBS StorageProvider = "obs"
	StorageProviderOSS StorageProvider = "oss"
)

type Config struct {
	App        string
	Provider   string
	Endpoint   string
	Region     string
	AccessKey  string
	SecretKey  string
	BucketName Bucket
}

type Bucket string
