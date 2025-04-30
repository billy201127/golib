package notify

import (
	"context"
	"log"
)

// Example 使用示例
func Example() {
	ctx := context.Background()

	// 创建钉钉通知
	dingTalkConfig := NotificationConfig{
		Type: DingTalk,
		Config: Config{
			Webhook: "https://oapi.dingtalk.com/robot/send?access_token=bbaa664f05f10a0d6c6f14022209a31f91caf26d0a7a3768c8b86b0b70855551",
			Secret:  "SEC1fc4f39463764053f1c5476566da6dadd57b81fe4a4c973a99cf1b54a933dbaa",
		},
	}
	dingTalkNotifier, err := NewNotification(dingTalkConfig)
	if err != nil {
		log.Fatalf("Failed to create dingtalk notifier: %v", err)
	}

	// 发送钉钉文本消息
	err = dingTalkNotifier.SendText(ctx, "这是一条钉钉测试消息")
	if err != nil {
		log.Printf("Failed to send dingtalk text message: %v", err)
	}

	// 发送钉钉卡片消息
	// err = dingTalkNotifier.SendCard(ctx, "钉钉卡片标题", "这是一条钉钉卡片消息内容\ntest\n", AtAll())
	// if err != nil {
	// 	log.Printf("Failed to send dingtalk card message: %v", err)
	// }

	// // 创建飞书通知
	// feishuConfig := NotificationConfig{
	// 	Type: Feishu,
	// 	Feishu: FeishuConfig{
	// 		Webhook: "your-feishu-webhook",
	// 		Secret:  "your-feishu-secret",
	// 	},
	// }
	// feishuNotifier, err := NewNotification(feishuConfig)
	// if err != nil {
	// 	log.Fatalf("Failed to create feishu notifier: %v", err)
	// }

	// // 发送飞书文本消息
	// err = feishuNotifier.SendText(ctx, "这是一条飞书测试消息", false, nil)
	// if err != nil {
	// 	log.Printf("Failed to send feishu text message: %v", err)
	// }

	// // 发送飞书卡片消息
	// err = feishuNotifier.SendCard(ctx, "飞书卡片标题", "这是一条飞书卡片消息内容", false)
	// if err != nil {
	// 	log.Printf("Failed to send feishu card message: %v", err)
	// }
}
