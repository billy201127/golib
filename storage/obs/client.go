package obs

import (
	"context"
	"fmt"
	"io"

	storagetypes "golib/storage/types"

	huaweiObs "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/zeromicro/go-zero/core/logc"
)

type Client struct {
	AppId     string
	obsClient *huaweiObs.ObsClient
}

func NewClient(ak, sk, endPoint string, appId string) (*Client, error) {
	obsClient, err := huaweiObs.New(ak, sk, endPoint)
	if err != nil {
		return nil, fmt.Errorf("Create obsClient error, errMsg: %s", err.Error())
	}

	return &Client{obsClient: obsClient, AppId: appId}, nil
}

func (c *Client) UploadFile(ctx context.Context, bucket storagetypes.Bucket, remote, local string) error {
	input := &huaweiObs.PutFileInput{}
	input.Bucket = string(bucket)
	input.Key = fmt.Sprintf("%s/%s", c.AppId, remote)
	input.SourceFile = local

	_, err := c.obsClient.PutFile(input)
	if err != nil {
		logc.Errorf(ctx, "Upload file error, errMsg: %s", err.Error())
	}

	return err
}

func (c *Client) UploadStream(ctx context.Context, bucket storagetypes.Bucket, remote string, stream io.Reader) error {
	input := &huaweiObs.PutObjectInput{}
	input.Bucket = string(bucket)
	input.Key = fmt.Sprintf("%s/%s", c.AppId, remote)
	input.Body = stream

	_, err := c.obsClient.PutObject(input)
	if err != nil {
		logc.Errorf(ctx, "Upload file error, errMsg: %s", err.Error())
	}

	return err
}

func (c *Client) DownloadFile(ctx context.Context, bucket storagetypes.Bucket, remote, local string) error {
	input := &huaweiObs.DownloadFileInput{}
	input.Bucket = string(bucket)
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

func (c *Client) DownloadStream(ctx context.Context, bucket storagetypes.Bucket, remote string) (io.ReadCloser, error) {
	input := &huaweiObs.GetObjectInput{}
	input.Bucket = string(bucket)
	input.Key = fmt.Sprintf("%s/%s", c.AppId, remote)

	output, err := c.obsClient.GetObject(input)
	if err != nil {
		logc.Errorf(ctx, "Download file error, errMsg: %s", err.Error())
	}
	defer output.Body.Close()

	return output.Body, err
}
