package main

import (
	"context"

	"gomod.pri/golib/xutils/watermark"
)

func main() {
	watermark.AddFromBytes(context.Background(), []byte(""), "test")
}
