package xhttp

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// DefaultTransport 默认的HTTP传输配置
var DefaultTransport = &http.Transport{
	MaxIdleConns:        500,
	MaxIdleConnsPerHost: 200,
	DialContext: (&net.Dialer{
		Timeout:   90 * time.Second,
		KeepAlive: 90 * time.Second,
	}).DialContext,
	TLSClientConfig: &tls.Config{
		ClientSessionCache: tls.NewLRUClientSessionCache(64),
	},
	ForceAttemptHTTP2:     true,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   90 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

// ClientOption 定义客户端配置选项
type ClientOption func(*Client)

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.client.Timeout = timeout
	}
}

// WithTransport 设置自定义Transport
func WithTransport(transport http.RoundTripper) ClientOption {
	return func(c *Client) {
		c.client.Transport = transport
	}
}

// WithHTTPClient 设置自定义HTTP客户端
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.client = client
	}
}

// Client HTTP客户端封装
type Client struct {
	client *http.Client
}

// NewClient 创建新的HTTP客户端
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		client: &http.Client{
			Transport: DefaultTransport,
			Timeout:   30 * time.Second, // 默认30秒超时
		},
	}

	// 应用选项
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Get 发送GET请求
func (c *Client) Get(ctx context.Context, url string, header map[string]string) (*http.Response, error) {
	return c.Do(ctx, http.MethodGet, url, header, nil)
}

// Post 发送POST请求
func (c *Client) Post(ctx context.Context, url string, header map[string]string, body []byte) (*http.Response, error) {
	return c.Do(ctx, http.MethodPost, url, header, body)
}

// Put 发送PUT请求
func (c *Client) Put(ctx context.Context, url string, header map[string]string, body []byte) (*http.Response, error) {
	return c.Do(ctx, http.MethodPut, url, header, body)
}

// Delete 发送DELETE请求
func (c *Client) Delete(ctx context.Context, url string, header map[string]string) (*http.Response, error) {
	return c.Do(ctx, http.MethodDelete, url, header, nil)
}

// Do 执行HTTP请求
func (c *Client) Do(ctx context.Context, method string, url string, header map[string]string, body []byte) (*http.Response, error) {
	var req *http.Request
	var err error

	if len(body) > 0 {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 添加链路追踪
	tracer := trace.TracerFromContext(req.Context())
	propagator := otel.GetTextMapPropagator()

	spanName := fmt.Sprintf("%s %s", method, req.URL.Path)
	ctx, span := tracer.Start(
		req.Context(),
		spanName,
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(semconv.HTTPClientAttributesFromHTTPRequest(req)...),
	)
	defer span.End()

	req = req.WithContext(ctx)
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

	// 设置请求头
	for k, v := range header {
		req.Header.Set(k, v)
	}

	// 执行请求
	resp, err := c.client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("execute request failed: %w", err)
	}

	// 设置追踪属性
	span.SetAttributes(semconv.HTTPAttributesFromHTTPStatusCode(resp.StatusCode)...)
	span.SetStatus(semconv.SpanStatusFromHTTPStatusCodeAndSpanKind(resp.StatusCode, oteltrace.SpanKindClient))

	if resp.StatusCode >= 400 {
		err = fmt.Errorf("http status %d", resp.StatusCode)
	}

	return resp, err
}

// GetClient 获取原始的http.Client
func (c *Client) GetClient() *http.Client {
	return c.client
}
