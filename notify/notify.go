package notify

import (
	"context"
	"fmt"
)

// NotificationType 通知类型
type NotificationType string

const (
	// DingTalk 钉钉通知
	DingTalk NotificationType = "dingtalk"
	// Feishu 飞书通知
	Feishu NotificationType = "feishu"
)

// NotificationConfig 通知配置
type NotificationConfig struct {
	Type   NotificationType // 通知类型
	Config Config           // 通知配置
}

type Config struct {
	Webhook string // 机器人 webhook
	Secret  string // 机器人加签密钥
}

// Notification 通知接口
type Notification interface {
	// SendText 发送文本消息
	SendText(ctx context.Context, content string, opts ...Option) error
	// SendCard 发送卡片消息
	SendCard(ctx context.Context, title, content string, opts ...Option) error
}

// Option 选项
type Option func(*Options)

// Options 选项结构
type Options struct {
	AtUsers []string // 空数组表示不@任何人，["all"]表示@所有人，["user1", "user2"]表示@特定用户
}

// AtAll 设置@所有人
func AtAll() Option {
	return func(o *Options) {
		o.AtUsers = []string{"all"}
	}
}

// AtMobiles 设置@指定手机号
func AtMobiles(atMobiles []string) Option {
	return func(o *Options) {
		o.AtUsers = atMobiles
	}
}

// NewNotification 创建通知实例
func NewNotification(cfg NotificationConfig) (Notification, error) {
	switch cfg.Type {
	case DingTalk:
		return NewDingTalkNotification(cfg.Config)
	case Feishu:
		return NewFeishuNotification(cfg.Config)
	default:
		return nil, fmt.Errorf("unsupported notification type: %s", cfg.Type)
	}
}
