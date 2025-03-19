package obs

import (
	"context"
	"fmt"
	"io"

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

	return &Client{obsClient: obsClient, AppId: cfg.App, bucket: cfg.BucketName}, nil
}

func (c *Client) UploadFile(ctx context.Context, remote, local string) error {
	input := &huaweiObs.PutFileInput{}
	input.Bucket = string(c.bucket)
	input.Key = fmt.Sprintf("%s/%s", c.AppId, remote)
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
	input.Key = fmt.Sprintf("%s/%s", c.AppId, remote)
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
	input.Key = fmt.Sprintf("%s/%s", c.AppId, remote)
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
	input.Key = fmt.Sprintf("%s/%s", c.AppId, remote)

	output, err := c.obsClient.GetObject(input)
	if err != nil {
		logc.Errorf(ctx, "Download file error, errMsg: %s", err.Error())
	}
	defer output.Body.Close()

	return output.Body, err
}

func (c *Client) SignUrl(ctx context.Context, remote string, expires int) (string, error) {
	input := &huaweiObs.CreateSignedUrlInput{
		Bucket:  string(c.bucket),
		Key:     fmt.Sprintf("%s/%s", c.AppId, remote),
		Expires: expires,
	}

	output, err := c.obsClient.CreateSignedUrl(input)
	if err != nil {
		logc.Errorf(ctx, "Create signed url error: %v", err)
		return "", err
	}

	return output.SignedUrl, nil
}
