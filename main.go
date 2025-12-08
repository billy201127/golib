package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	"gomod.pri/golib/xutils/logutil"
	"gomod.pri/golib/xutils/watermark"
)

func main1() {
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

func main() {
	logx.MustSetup(logx.LogConf{
		ServiceName: "test",
		Mode:        "console",
		Encoding:    "plain",
		Level:       "debug",
		Compress:    false,
		Stat:        false,
	})

	hookWriter := logutil.NewHookWriter(os.Stdout, logutil.Config{
		IntervalSec:    1,
		Limit:          10,
		DisableStmtLog: false,
		NotifyWebhook:  "https://oapi.dingtalk.com/robot/send?access_token=bafacc990f3c25c324f7faae7e095adb4054fe254c15bc8e5f48732591e130df",
		NotifySecret:   "SECa0c8a83a2c2e3c4f3080ba1f5fcd855d24b7fa8e2cf5434463162eed66a21a7c",
	})

	logx.SetWriter(logx.NewWriter(hookWriter))

	logx.Error("mervyn test")
	logx.Error("mervyn test01")

	// 使用goroutine来触发panic，避免主程序退出
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logx.Errorf("mervyn test03: 捕获到panic: %v", err)
			}
		}()

		// 在goroutine中触发panic
		panic("mervyn test02: {'code': 400, 'message': 'test'}\ntest01\ntest02")
	}()

	// 主程序继续运行20秒
	fmt.Println("程序继续运行，等待20秒...")
	time.Sleep(20 * time.Second)
	fmt.Println("程序正常退出")
}
