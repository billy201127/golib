package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"golib/storage/obs"
	"golib/storage/oss"
	storagetypes "golib/storage/types"
)

type Storage interface {
	UploadFile(ctx context.Context, bucket storagetypes.Bucket, remote, local string) error
	UploadStream(ctx context.Context, bucket storagetypes.Bucket, remote string, stream io.Reader) error

	DownloadFile(ctx context.Context, bucket storagetypes.Bucket, remote, local string) error
	DownloadStream(ctx context.Context, bucket storagetypes.Bucket, remote string) (io.ReadCloser, error)
}

func NewStorage(appId string, cfg storagetypes.Config) (Storage, error) {
	provider := storagetypes.StorageProvider(strings.ToLower(cfg.Provider))

	switch provider {
	case storagetypes.StorageProviderOBS:
		return obs.NewClient(cfg.AccessKey, cfg.SecretKey, cfg.Endpoint, appId)
	case storagetypes.StorageProviderOSS:
		return oss.NewClient(cfg.Endpoint, appId, cfg.AccessKey, cfg.SecretKey)
	default:
		return nil, fmt.Errorf("Unsupported storage provider: %s", cfg.Provider)
	}
}
