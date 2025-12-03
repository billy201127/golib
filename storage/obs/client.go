package obs

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	huaweiObs "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/zeromicro/go-zero/core/logc"
	"gomod.pri/golib/storage/types"
)

type Client struct {
	AppId     string
	obsClient *huaweiObs.ObsClient
	bucket    types.Bucket
}

func NewClient(cfg types.Config) (*Client, error) {
	obsClient, err := huaweiObs.New(cfg.AccessKey, cfg.SecretKey, cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("Create obsClient error, errMsg: %s", err.Error())
	}

	return &Client{obsClient: obsClient, AppId: cfg.App, bucket: cfg.Bucket}, nil
}

// buildKey 构建完整的对象Key，避免双斜杠问题
func (c *Client) buildKey(remote string) string {
	// 移除remote开头的斜杠
	remote = strings.TrimPrefix(remote, "/")
	// 确保AppId不以斜杠结尾
	appId := strings.TrimSuffix(c.AppId, "/")
	// 构建完整路径
	if appId == "" {
		return remote
	}
	return fmt.Sprintf("%s/%s", appId, remote)
}

func (c *Client) UploadFile(ctx context.Context, remote, local string) error {
	input := &huaweiObs.PutFileInput{}
	input.Bucket = string(c.bucket)
	input.Key = c.buildKey(remote)
	input.SourceFile = local

	_, err := c.obsClient.PutFile(input)
	if err != nil {
		logc.Errorf(ctx, "Upload file error, errMsg: %s", err.Error())
	}

	return err
}

func (c *Client) UploadStream(ctx context.Context, remote string, stream io.Reader) error {
	input := &huaweiObs.PutObjectInput{}
	input.Bucket = string(c.bucket)
	input.Key = c.buildKey(remote)
	input.Body = stream

	_, err := c.obsClient.PutObject(input)
	if err != nil {
		logc.Errorf(ctx, "Upload file error, errMsg: %s", err.Error())
	}

	return err
}

func (c *Client) DownloadFile(ctx context.Context, remote, local string) error {
	input := &huaweiObs.DownloadFileInput{}
	input.Bucket = string(c.bucket)
	input.Key = c.buildKey(remote)
	input.DownloadFile = local

	input.EnableCheckpoint = true
	input.PartSize = 10 * 1024 * 1024
	input.TaskNum = 5

	_, err := c.obsClient.DownloadFile(input)
	if err != nil {
		logc.Errorf(ctx, "Download file error, errMsg: %s", err.Error())
	}

	return err
}

func (c *Client) DownloadStream(ctx context.Context, remote string) (io.ReadCloser, error) {
	input := &huaweiObs.GetObjectInput{}
	input.Bucket = string(c.bucket)
	input.Key = c.buildKey(remote)

	output, err := c.obsClient.GetObject(input)
	if err != nil {
		logc.Errorf(ctx, "Download file error, errMsg: %s", err.Error())
		return nil, err
	}

	return output.Body, err
}

func (c *Client) SignUrl(ctx context.Context, remote string, expires int) (string, error) {
	// 构建Key，避免双斜杠问题
	key := c.buildKey(remote)

	input := &huaweiObs.CreateSignedUrlInput{
		Method:  huaweiObs.HttpMethodGet,
		Bucket:  string(c.bucket),
		Key:     key,
		Expires: expires,
	}

	output, err := c.obsClient.CreateSignedUrl(input)
	if err != nil {
		logc.Errorf(ctx, "Create signed url error: %v, key: %s", err, key)
		return "", err
	}

	if output.SignedUrl == "" {
		return "", fmt.Errorf("Signed url is empty")
	}

	return url.QueryEscape(output.SignedUrl), nil
}

func (c *Client) CopyFile(ctx context.Context, source, target string) error {
	input := &huaweiObs.CopyObjectInput{
		ObjectOperationInput: huaweiObs.ObjectOperationInput{
			Bucket: string(c.bucket),
			Key:    c.buildKey(target),
		},
		CopySourceBucket: string(c.bucket),
		CopySourceKey:    c.buildKey(source),
	}

	_, err := c.obsClient.CopyObject(input)
	if err != nil {
		logc.Errorf(ctx, "Copy file error, errMsg: %s", err.Error())
	}

	return err
}
