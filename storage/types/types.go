package types

type StorageProvider string

const (
	StorageProviderOBS StorageProvider = "obs"
	StorageProviderOSS StorageProvider = "oss"
	StorageProviderS3  StorageProvider = "s3"
)

type Config struct {
	App       string
	Provider  string
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    Bucket
}

type Bucket string
