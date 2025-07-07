package apollo

import (
	"fmt"

	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
)

type Config struct {
	AppID           string
	Cluster         string
	Addr            string
	PrivateSpace    string
	ChangeListeners []storage.ChangeListener // 支持多个配置变更监听器
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

	// add change listeners if provided
	if len(conf.ChangeListeners) > 0 {
		for _, listener := range conf.ChangeListeners {
			client.AddChangeListener(listener)
		}
	} else {
		// use default listener if no listeners provided
		client.AddChangeListener(&CustomChangeListener{})
	}

	c := &Client{
		client:  &client,
		Default: client.GetConfig(ApplicationNamespace),
		Private: client.GetConfig(conf.PrivateSpace),
	}

	return c, nil
}

// AddChangeListener 向已存在的客户端添加新的配置变更监听器
func (c *Client) AddChangeListener(listener storage.ChangeListener) {
	if c.client != nil {
		(*c.client).AddChangeListener(listener)
	}
}

// CustomChangeListener 默认的配置变更监听器
type CustomChangeListener struct{}

func (c *CustomChangeListener) OnChange(event *storage.ChangeEvent) {
	// logx.Infof("Apollo Config Changed: %v\n", event.Changes)
}

func (c *CustomChangeListener) OnNewestChange(event *storage.FullChangeEvent) {
	// logx.Infof("Apollo Config Full Update: %v\n", event.Changes)
}
