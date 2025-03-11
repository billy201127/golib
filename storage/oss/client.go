package oss

import (
	"context"
	"fmt"
	"io"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	aliOss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/zeromicro/go-zero/core/logc"
	storagetypes "gomod.pri/golib/storage/types"
)

type Client struct {
	AppId     string
	ossClient *aliOss.Client
}

func NewClient(region string, appId string, ak, sk string) (*Client, error) {
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(ak, sk)).
		WithRegion(region)

	client := oss.NewClient(cfg)

	return &Client{ossClient: client, AppId: appId}, nil
}

func (c *Client) UploadFile(ctx context.Context, bucket storagetypes.Bucket, remote, local string) error {
	_, err := c.ossClient.PutObjectFromFile(ctx, &oss.PutObjectRequest{
		Bucket: oss.Ptr(string(bucket)),
		Key:    oss.Ptr(fmt.Sprintf("%s/%s", c.AppId, remote)),
	}, local)
	if err != nil {
		logc.Errorf(ctx, "Upload file error, errMsg: %s", err.Error())
	}

	return err
}

func (c *Client) UploadStream(ctx context.Context, bucket storagetypes.Bucket, remote string, stream io.Reader) error {
	request := &oss.PutObjectRequest{
		Bucket: oss.Ptr(string(bucket)),
		Key:    oss.Ptr(fmt.Sprintf("%s/%s", c.AppId, remote)),
		Body:   stream,
	}

	_, err := c.ossClient.PutObject(ctx, request)
	if err != nil {
		logc.Errorf(ctx, "Upload stream error, errMsg: %s", err.Error())
	}

	return err
}

func (c *Client) DownloadFile(ctx context.Context, bucket storagetypes.Bucket, remote, local string) error {
	_, err := c.ossClient.GetObjectToFile(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(string(bucket)),
		Key:    oss.Ptr(fmt.Sprintf("%s/%s", c.AppId, remote)),
	}, local)
	if err != nil {
		logc.Errorf(ctx, "Download file error, errMsg: %s", err.Error())
	}

	return err
}

func (c *Client) DownloadStream(ctx context.Context, bucket storagetypes.Bucket, remote string) (io.ReadCloser, error) {
	request := &oss.GetObjectRequest{
		Bucket: oss.Ptr(string(bucket)),
		Key:    oss.Ptr(fmt.Sprintf("%s/%s", c.AppId, remote)),
	}
	result, err := c.ossClient.GetObject(ctx, request)
	if err != nil {
		logc.Errorf(ctx, "Download stream error, errMsg: %s", err.Error())
	}
	defer result.Body.Close()

	return result.Body, err
}
