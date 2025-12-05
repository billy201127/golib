//go:build !cgo
// +build !cgo

package watermark

import (
	"context"
	"errors"
	"io"

	"github.com/zeromicro/go-zero/core/logc"
)

// Add support watermark without cgo
func Add(ctx context.Context, path string, text string) (io.ReadCloser, error) {
	logc.Errorf(ctx, "watermark need cgo support, please compile with CGO_ENABLED=1")
	return nil, errors.New("watermark func Add is not implemented without cgo")
}
