package rocketmq

import (
	"os"

	rmq "github.com/apache/rocketmq-clients/golang/v5"
)

func SetLogger() {
	os.Setenv(rmq.CLIENT_LOG_ROOT, "./rocketmqlogs")
	os.Setenv(rmq.ENABLE_CONSOLE_APPENDER, "true")
	os.Setenv(rmq.CLIENT_LOG_LEVEL, "warn")
	rmq.ResetLogger()
}
