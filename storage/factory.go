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
	UploadFile(ctx context.Context, remote, local string) error
	UploadStream(ctx context.Context, remote string, stream io.Reader) error

	DownloadFile(ctx context.Context, remote, local string) error
	DownloadStream(ctx context.Context, remote string) (io.ReadCloser, error)

	SignUrl(ctx context.Context, remote string, expires int) (string, error)
}

func NewStorage(appId string, cfg types.Config) (Storage, error) {
	provider := types.StorageProvider(strings.ToLower(cfg.Provider))

	switch provider {
	case types.StorageProviderOBS:
		return obs.NewClient(cfg)
	case types.StorageProviderOSS:
		return oss.NewClient(cfg)
	default:
		return nil, fmt.Errorf("Unsupported storage provider: %s", cfg.Provider)
	}
}
