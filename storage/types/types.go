package types

type StorageProvider string

const (
	StorageProviderOBS StorageProvider = "obs"
	StorageProviderOSS StorageProvider = "oss"
)

type Config struct {
	Provider  string
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
}

type Bucket string
