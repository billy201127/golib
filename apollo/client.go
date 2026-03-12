package apollo

import (
	"fmt"
	"strings"

	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
	"gopkg.in/yaml.v3"
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

func (c *Client) GetPrivateJson() []byte {
	content := strings.TrimPrefix(c.Private.GetContent(), "content=")
	return []byte(content)
}

// GetPrivateYamlFromProperties 将 properties 风格内容还原为 YAML（功能等价）
func (c *Client) GetPrivateYaml() []byte {
	content := strings.TrimPrefix(c.Private.GetContent(), "content=")
	props := parsePropertiesInline(content)
	data := buildNestedMap(props)
	out, err := yaml.Marshal(data)
	if err != nil {
		return []byte(content)
	}
	return out
}

// AddChangeListener 向已存在的客户端添加新的配置变更监听器
func (c *Client) AddChangeListener(listener storage.ChangeListener) {
	if c.client != nil {
		(*c.client).AddChangeListener(listener)
	}
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

	c := &Client{
		client:  &client,
		Default: client.GetConfig(ApplicationNamespace),
		Private: client.GetConfig(conf.PrivateSpace),
	}

	return c, nil
}

// CustomChangeListener 默认的配置变更监听器
type CustomChangeListener struct{}

func (c *CustomChangeListener) OnChange(event *storage.ChangeEvent) {
	// logx.Infof("Apollo Config Changed: %v\n", event.Changes)
}

func (c *CustomChangeListener) OnNewestChange(event *storage.FullChangeEvent) {
	// logx.Infof("Apollo Config Full Update: %v\n", event.Changes)
}
