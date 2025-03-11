package apollo

import (
	"fmt"

	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
	"github.com/zeromicro/go-zero/core/logx"
)

type Config struct {
	AppID        string
	Cluster      string
	Addr         string
	PrivateSpace string
}

// Client Apollo 客户端封装
type Client struct {
	client  *agollo.Client
	Default *storage.Config // application namespace
	Private *storage.Config // private namespace
}

// 预定义的命名空间
var (
	ApplicationNamespace = "application"
)

func NewClient(conf *Config) (*Client, error) {
	client, err := agollo.StartWithConfig(func() (*config.AppConfig, error) {
		return &config.AppConfig{
			AppID:          conf.AppID,
			Cluster:        conf.Cluster,
			NamespaceName:  ApplicationNamespace,
			IP:             conf.Addr,
			IsBackupConfig: true,
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("create apollo client error: %w", err)
	}

	// add change listener
	client.AddChangeListener(&CustomChangeListener{})

	c := &Client{
		client:  &client,
		Default: client.GetConfig(ApplicationNamespace),
		Private: client.GetConfig(conf.PrivateSpace),
	}

	return c, nil
}

type CustomChangeListener struct{}

func (c *CustomChangeListener) OnChange(event *storage.ChangeEvent) {
	logx.Infof("Apollo Config Changed: %v\n", event.Changes)
}

func (c *CustomChangeListener) OnNewestChange(event *storage.FullChangeEvent) {
	logx.Infof("Apollo Config Full Update: %v\n", event.Changes)
}
