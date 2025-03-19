package oss

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	aliOss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/zeromicro/go-zero/core/logc"
	"gomod.pri/golib/storage/types"
)

type Client struct {
	AppId     string
	ossClient *aliOss.Client
	bucket    types.Bucket
}

func NewClient(cfg types.Config) (*Client, error) {
	config := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey)).
		WithEndpoint(cfg.Endpoint).
		WithRegion(cfg.Region)

	client := oss.NewClient(config)
	return &Client{ossClient: client, AppId: cfg.App, bucket: cfg.Bucket}, nil
}

func (c *Client) UploadFile(ctx context.Context, remote, local string) error {
	_, err := c.ossClient.PutObjectFromFile(ctx, &oss.PutObjectRequest{
		Bucket: oss.Ptr(string(c.bucket)),
		Key:    oss.Ptr(fmt.Sprintf("%s/%s", c.AppId, remote)),
	}, local)
	if err != nil {
		logc.Errorf(ctx, "Upload file error, errMsg: %s", err.Error())
	}

	return err
}

func (c *Client) UploadStream(ctx context.Context, remote string, stream io.Reader) error {
	request := &oss.PutObjectRequest{
		Bucket: oss.Ptr(string(c.bucket)),
		Key:    oss.Ptr(fmt.Sprintf("%s/%s", c.AppId, remote)),
		Body:   stream,
	}

	_, err := c.ossClient.PutObject(ctx, request)
	if err != nil {
		logc.Errorf(ctx, "Upload stream error, errMsg: %s", err.Error())
	}

	return err
}

func (c *Client) DownloadFile(ctx context.Context, remote, local string) error {
	_, err := c.ossClient.GetObjectToFile(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(string(c.bucket)),
		Key:    oss.Ptr(fmt.Sprintf("%s/%s", c.AppId, remote)),
	}, local)
	if err != nil {
		logc.Errorf(ctx, "Download file error, errMsg: %s", err.Error())
	}

	return err
}

func (c *Client) DownloadStream(ctx context.Context, remote string) (io.ReadCloser, error) {
	request := &oss.GetObjectRequest{
		Bucket: oss.Ptr(string(c.bucket)),
		Key:    oss.Ptr(fmt.Sprintf("%s/%s", c.AppId, remote)),
	}
	result, err := c.ossClient.GetObject(ctx, request)
	if err != nil {
		logc.Errorf(ctx, "Download stream error, errMsg: %s", err.Error())
	}
	defer result.Body.Close()

	return result.Body, err
}

func (c *Client) SignUrl(ctx context.Context, remote string, expires int) (string, error) {
	req, err := c.ossClient.Presign(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(string(c.bucket)),
		Key:    oss.Ptr(fmt.Sprintf("%s/%s", c.AppId, remote)),
	}, oss.PresignExpires(time.Second*time.Duration(expires)))
	if err != nil {
		logc.Errorf(ctx, "Sign url error, errMsg: %s", err.Error())
	}

	return req.URL, nil
}
