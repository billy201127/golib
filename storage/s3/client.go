package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gomod.pri/golib/storage/types"
)

type Client struct {
	s3Client *s3.Client
	bucket   string
	AppId    string
}

func NewClient(cfg types.Config) (*Client, error) {
	// load aws config
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cfg.AccessKey,
				SecretAccessKey: cfg.SecretKey,
			}, nil
		})),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	// create s3 client
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // use path style for s3, default is virtual hosted-style
	})

	return &Client{
		s3Client: s3Client,
		bucket:   string(cfg.Bucket),
		AppId:    cfg.App,
	}, nil
}

func (c *Client) UploadFile(ctx context.Context, remote, local string) error {
	file, err := os.Open(local)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	return c.UploadStream(ctx, remote, file)
}

func (c *Client) UploadStream(ctx context.Context, remote string, stream io.Reader) error {
	key := fmt.Sprintf("%s/%s", c.AppId, remote)

	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   stream,
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

func (c *Client) DownloadFile(ctx context.Context, remote, local string) error {
	// ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(local), 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// create local file
	file, err := os.Create(local)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	// get file stream
	stream, err := c.DownloadStream(ctx, remote)
	if err != nil {
		return err
	}
	defer stream.Close()

	// copy content
	_, err = io.Copy(file, stream)
	if err != nil {
		return fmt.Errorf("failed to copy content to local file: %w", err)
	}

	return nil
}

func (c *Client) DownloadStream(ctx context.Context, remote string) (io.ReadCloser, error) {
	key := fmt.Sprintf("%s/%s", c.AppId, remote)

	result, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	return result.Body, nil
}

func (c *Client) SignUrl(ctx context.Context, remote string, expires int) (string, error) {
	key := fmt.Sprintf("%s/%s", c.AppId, remote)

	presignClient := s3.NewPresignClient(c.s3Client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return request.URL, nil
}
