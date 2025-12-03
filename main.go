package main

import (
	"context"
	"fmt"
	"time"

	"gomod.pri/golib/xutils/watermark"
)

func main() {
	now := time.Now()
	watermark.Add(context.Background(), "https://allproject-test-oss.oss-ap-southeast-1.aliyuncs.com/Sahara/input/6b/d2eb/6bd2ebb6551d789a8d0e05856ab0bdbd.jpg?x-oss-credential=LTAI5tPGy5cmCzrCd2H8i7eR%2F20251203%2Fap-southeast-1%2Foss%2Faliyun_v4_request&x-oss-date=20251203T125019Z&x-oss-expires=1800&x-oss-signature=6c506b4a324d29b7a1d2585777a4d41add63502f1b955e1769afe9bd22c8eceb&x-oss-signature-version=OSS4-HMAC-SHA256", "testtest")
	fmt.Println(time.Since(now))
}
