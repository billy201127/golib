package kmscred

type Vendor string
type Mode string

const (
	VendorAliyun      Vendor = "aliyun"
	VendorHuaweiCloud Vendor = "huaweicloud"
	VendorAWS         Vendor = "aws"

	ModeAKSK Mode = "aksk"
	ModeRAM  Mode = "ram"
)

type Config struct {
	Vendor    Vendor
	Mode      Mode
	AccessKey string
	SecretKey string
	Region    string
	Extra     map[string]string
}

// SecretInfo represents secret information returned by KMS
type SecretInfo struct {
	Name  string
	Value string
}
