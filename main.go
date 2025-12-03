package main

import (
	"fmt"

	"github.com/goccy/go-yaml"
	"gomod.pri/golib/apollo"
)

func main() {
	client, err := apollo.NewClient(&apollo.Config{
		AppID:        "Debt",
		Cluster:      "default",
		Addr:         "http://config.apollo.host:8080",
		PrivateSpace: "test.yaml",
	})
	if err != nil {
		panic(err)
	}

	content := client.Private.GetContent()
	fmt.Println(content)

	var config DebtConfig
	err = yaml.Unmarshal([]byte(content), &config)
	if err != nil {
		panic(err)
	}

	fmt.Println(config)
}

type DebtConfig struct {
	LogHook Config
}

type Config struct {
	IntervalSec    int64  `yaml:"IntervalSec"`
	Limit          int    `yaml:"Limit"`
	DisableStmtLog bool   `yaml:"DisableStmtLog"`
	NotifyWebhook  string `yaml:"NotifyWebhook"`
	NotifySecret   string `yaml:"NotifySecret"`
}
