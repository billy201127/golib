package notify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"gomod.pri/golib/xhttp"
)

// DingTalkNotification 钉钉通知实现
type DingTalkNotification struct {
	webhook string
	secret  string
}

// NewDingTalkNotification 创建钉钉通知实例
func NewDingTalkNotification(cfg Config) (Notification, error) {
	if cfg.Webhook == "" {
		return nil, fmt.Errorf("webhook is empty")
	}
	return &DingTalkNotification{
		webhook: cfg.Webhook,
		secret:  cfg.Secret,
	}, nil
}

// SendText 发送文本消息
func (d *DingTalkNotification) SendText(ctx context.Context, content string, isAtAll bool, atMobiles []string) error {
	return d.sendDingTalkTextMsg(ctx, content, atMobiles, isAtAll)
}

// SendCard 发送卡片消息
func (d *DingTalkNotification) SendCard(ctx context.Context, title, content string, isAtAll bool) error {
	return d.sendDingTalkMarkdownMsg(ctx, title, content, isAtAll)
}

// 生成钉钉签名
func (d *DingTalkNotification) GenDingTalkSign() (string, int64) {
	timestamp := time.Now().UnixMilli()
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, d.secret)
	h := hmac.New(sha256.New, []byte(d.secret))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature, timestamp
}

// 发送text格式钉钉消息
func (d *DingTalkNotification) sendDingTalkTextMsg(ctx context.Context, content string, mobiles []string, isAtAll bool) (err error) {
	msg := &Dtext{}
	msg.Msgtype = "text"
	msg.Text.Content = content
	msg.At.AtMobiles = mobiles
	msg.At.IsAtAll = isAtAll
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	err = d.sendDingTalkMsg(ctx, string(data))
	return
}

// 发送markdown格式钉钉消息
func (d *DingTalkNotification) sendDingTalkMarkdownMsg(ctx context.Context, title, content string, isAtAll bool) (err error) {
	msg := &Dmarkdown{}
	msg.Msgtype = "markdown"
	msg.Markdown.Title = title
	msg.Markdown.Text = content
	msg.At.IsAtAll = isAtAll
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	err = d.sendDingTalkMsg(ctx, string(data))
	return
}

// 发送钉钉消息
func (d *DingTalkNotification) sendDingTalkMsg(ctx context.Context, reqBody string) (err error) {
	if strings.TrimSpace(d.webhook) == "" {
		err = fmt.Errorf("webhook is empty")
		return
	}

	// 构建请求URL
	robotUrl := d.webhook

	// 如果设置了签名密钥，则添加签名参数
	if d.secret != "" {
		sign, timestamp := d.GenDingTalkSign()
		robotUrl = fmt.Sprintf("%s&timestamp=%d&sign=%s", d.webhook, timestamp, sign)
	}

	reqHeaders := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := xhttp.NewClient().Post(ctx, robotUrl, reqHeaders, []byte(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	var resData TalkResponse
	err = json.Unmarshal(body, &resData)
	if err != nil {
		return
	}
	if resData.Code != 0 {
		err = fmt.Errorf("%s", resData.Msg)
	}
	return
}

// 钉钉消息结构体
// text类型
type Dtext struct {
	Msgtype string  `json:"msgtype"` //消息类型，此时固定为：text
	Text    Content `json:"text"`
	At      At      `json:"at"`
}

type Content struct {
	Content string `json:"content"` //消息内容
}

type At struct {
	AtMobiles []string `json:"atMobiles"` //被@人的手机号(在content里添加@人的手机号)
	IsAtAll   bool     `json:"isAtAll"`   //@所有人时：true，否则为：false
}

// markdown类型
type Dmarkdown struct {
	Msgtype  string   `json:"msgtype"`  //消息类型，此时固定为：markdown
	Markdown Markdown `json:"markdown"` //markdown消息
	At       At       `json:"at"`       //@信息
}

type Markdown struct {
	Title string `json:"title"` //标题
	Text  string `json:"text"`  //markdown格式的消息内容
}

type TalkResponse struct {
	Code int    `json:"errcode"`
	Msg  string `json:"errmsg"`
}
