//go:build !cgo
// +build !cgo

package watermark

import (
	"context"
	"errors"
	"io"

	"github.com/zeromicro/go-zero/core/logc"
)

// Add 在没有cgo支持时的占位实现
func Add(ctx context.Context, path string, text string) (io.ReadCloser, error) {
	logc.Errorf(ctx, "watermark func Add is not implemented without cgo, please compile with CGO_ENABLED=1")
	return nil, errors.New("watermark func Add is not implemented without cgo")
}
