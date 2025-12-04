package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"gomod.pri/golib/xutils/watermark"
)

func main() {
	url := "http://s3.rapidcompute.com/pidn-oss-sahara-image/Sahara/input/38/bd58/38bd589237b8ba38421f29abfebe4eb6.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Checksum-Mode=ENABLED&X-Amz-Credential=rcus_awamifin/20251204/us-east-1/s3/aws4_request&X-Amz-Date=20251204T073925Z&X-Amz-Expires=900&X-Amz-SignedHeaders=host&x-id=GetObject&X-Amz-Signature=d5d3df99debae59ec3f9ec9b21acf11e7d7150b322faa1689a4cc3a484f1cbfb"
	now := time.Now()
	data, err := watermark.Add(context.Background(), url, "test")
	if err != nil {
		logc.Errorf(context.Background(), "watermark.Add error: %v", err)
		return
	}
	fmt.Println(time.Since(now))

	// write to file
	content, err := io.ReadAll(data)
	if err != nil {
		logc.Errorf(context.Background(), "readAll error: %v", err)
		return
	}
	os.WriteFile("/Users/billy/Desktop/output.jpg", content, 0644)
}
