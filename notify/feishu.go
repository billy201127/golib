package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"gomod.pri/golib/xhttp"
)

// FeishuNotification 飞书通知实现
type FeishuNotification struct {
	webhook string
	secret  string
}

// NewFeishuNotification 创建飞书通知实例
func NewFeishuNotification(cfg Config) (Notification, error) {
	if cfg.Webhook == "" || cfg.Secret == "" {
		return nil, fmt.Errorf("feishu webhook or secret is empty")
	}
	return &FeishuNotification{
		webhook: cfg.Webhook,
		secret:  cfg.Secret,
	}, nil
}

// SendText 发送文本消息
func (f *FeishuNotification) SendText(ctx context.Context, content string, isAtAll bool, atMobiles []string) error {
	// 飞书不支持直接 @ 手机号，只支持 @ 所有人
	return SendFeishuTextMsg(ctx, f.webhook, f.secret, content, isAtAll)
}

// SendCard 发送卡片消息
func (f *FeishuNotification) SendCard(ctx context.Context, title, content string, isAtAll bool) error {
	return SendFeishuCardMsg(ctx, f.webhook, f.secret, title, content, isAtAll)
}

// 发送飞书文本消息
func SendFeishuTextMsg(ctx context.Context, webhook, secret, content string, isAtAll bool) error {
	if webhook == "" || secret == "" {
		return nil
	}
	tt := time.Now().Unix()
	secretStr, _ := GenFeishuSign(ctx, secret, tt)
	info := reqStruct{}
	info.MsgType = "text"
	info.Content.Text = content
	if isAtAll {
		info.Content.Text += `<at user_id="all">Everyone</at>`
	}
	info.Timestamp = strconv.FormatInt(tt, 10)
	info.Sign = secretStr
	dataB, _ := json.Marshal(info)
	header := map[string]string{
		"Content-Type": "application/json",
	}
	resp, err := xhttp.NewClient().Post(ctx, webhook, header, dataB)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return err
}

// 生成飞书签名
func GenFeishuSign(ctx context.Context, secret string, timestamp int64) (string, error) {
	//timestamp + key 做sha256, 再进行base64 encode
	stringToSign := fmt.Sprintf("%v", timestamp) + "\n" + secret
	var data []byte
	h := hmac.New(sha256.New, []byte(stringToSign))
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}

// 发送飞书卡片消息
func SendFeishuCardMsg(ctx context.Context, webhook, secret, title, content string, isAtAll bool) error {
	if webhook == "" || secret == "" {
		return fmt.Errorf("invalid config")
	}

	tt := time.Now().Unix()
	secretStr, _ := GenFeishuSign(ctx, secret, tt)
	msg := CardMsg{
		MsgType:   "interactive",
		Timestamp: strconv.FormatInt(tt, 10),
		Sign:      secretStr,
	}

	msg.Card.Config.EnableForward = true
	msg.Card.Config.WideScreenMode = true

	msg.Card.Header.Title.Tag = "plain_text"
	msg.Card.Header.Title.Content = title
	msg.Card.Header.Template = "blue"
	if isAtAll {
		msg.Card.Header.Template = "red"
		content += `<at id=all>Everyone</at>`
	}

	hostname, _ := os.Hostname()
	content = fmt.Sprintf("Hostname: [%s]\n%s\n", hostname, content)
	element := Element{
		Tag:     "markdown",
		Content: content,
	}
	msg.Card.Elements = append(msg.Card.Elements, element)

	data, _ := json.Marshal(msg)
	request, err := http.NewRequest("POST", webhook, bytes.NewReader(data))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	client := http.Client{
		Timeout: time.Second * 5,
	}
	_, err = client.Do(request)

	return err
}

// 飞书消息结构体
type reqStruct struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
	Timestamp string `json:"timestamp"`
	Sign      string `json:"sign"`
}

type CardMsg struct {
	MsgType   string `json:"msg_type"`
	Timestamp string `json:"timestamp"`
	Sign      string `json:"sign"`
	Card      struct {
		Config struct {
			WideScreenMode bool `json:"wide_screen_mode"`
			EnableForward  bool `json:"enable_forward"`
		} `json:"config"`
		Header struct {
			Template string `json:"template"`
			Title    struct {
				Tag     string `json:"tag"`
				Content string `json:"content"`
			} `json:"title"`
		}
		Elements []Element `json:"elements"`
	} `json:"card"`
}

type Element struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}
