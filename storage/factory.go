package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"gomod.pri/golib/storage/obs"
	"gomod.pri/golib/storage/oss"
	"gomod.pri/golib/storage/types"
)

type Storage interface {
	UploadFile(ctx context.Context, bucket types.Bucket, remote, local string) error
	UploadStream(ctx context.Context, bucket types.Bucket, remote string, stream io.Reader) error

	DownloadFile(ctx context.Context, bucket types.Bucket, remote, local string) error
	DownloadStream(ctx context.Context, bucket types.Bucket, remote string) (io.ReadCloser, error)
}

func NewStorage(appId string, cfg types.Config) (Storage, error) {
	provider := types.StorageProvider(strings.ToLower(cfg.Provider))

	switch provider {
	case types.StorageProviderOBS:
		return obs.NewClient(cfg.AccessKey, cfg.SecretKey, cfg.Endpoint, appId)
	case types.StorageProviderOSS:
		return oss.NewClient(cfg.Endpoint, cfg.Region, appId, cfg.AccessKey, cfg.SecretKey)
	default:
		return nil, fmt.Errorf("Unsupported storage provider: %s", cfg.Provider)
	}
}
