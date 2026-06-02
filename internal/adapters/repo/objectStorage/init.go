package objectstorage

import (
	"bytes"
	"context"
	"dpm/internal/config"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	aws_cfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"

	// "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	expired = time.Hour * 24

	maxBackoff  = time.Second * 12
	maxAttempts = 5
)

type S3Client struct {
	client     *s3.Client
	bucketName string
	downloader *manager.Downloader
}

func NewS3Client(ctx context.Context, c config.S3) (S3Client, error) {
	const op = "./internal/adapters/repository/objectStorage/init.go.NewS3Client()"

	slog.Info(os.Getenv("AWS_BUCKET"))

	accessKeyID := c.AccessKey
	accessSecretKey := c.SecretKey
	endpoint := c.Endpoint
	region := c.Region
	slog.Info("REGION: " + region)
	slog.Info(fmt.Sprintf("%+v", c))
	cfg, err := aws_cfg.LoadDefaultConfig(ctx,
		aws_cfg.WithRegion(region),
		aws_cfg.WithBaseEndpoint(endpoint),
		aws_cfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, accessSecretKey, "")),
	)
	if err != nil {
		return S3Client{}, fmt.Errorf("%s: %w", op, err)
	}

	client := s3.NewFromConfig(cfg,
		func(o *s3.Options) {
			o.Retryer = retry.NewAdaptiveMode(
				func(amo *retry.AdaptiveModeOptions) {
					amo.StandardOptions = []func(*retry.StandardOptions){
						func(so *retry.StandardOptions) {
							so.MaxAttempts = 5
							so.MaxBackoff = time.Second * 12
							so.Retryables = retry.DefaultRetryables
						},
					}
				},
			)
			o.UsePathStyle = true
		},
	)

	l, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return S3Client{}, fmt.Errorf("%s: %w", op, err)
	}

	for i := range l.Buckets {
		slog.Info(fmt.Sprintf("BUCKET: %v", aws.ToString(l.Buckets[i].Name)))
	}

	downloader := manager.NewDownloader(client, func(d *manager.Downloader) {
		d.PartBodyMaxRetries = 10
	})

	slog.Info("Return s3")

	return S3Client{
		client:     client,
		bucketName: os.Getenv("AWS_BUCKET"),
		downloader: downloader,
	}, nil
}

func (s3Client S3Client) UploadObject(ctx context.Context, key string, data []byte, contentType string) error {
	const op = "./internal/adapters/repository/objectStorage/init.go.UploadObject()"

	slog.Info(key)
	slog.Info(contentType)
	_, err := s3Client.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s3Client.bucketName,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s3Client S3Client) GetObject(ctx context.Context, key string, w io.WriterAt) error {
	const op = "./internal/adapters/repository/objectStorage/init.go.GetObject()"

	_, err := s3Client.downloader.Download(ctx, w, &s3.GetObjectInput{
		Bucket: &s3Client.bucketName,
		Key:    &key,
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s3Client S3Client) DeleteObject(ctx context.Context, key string) error {
	const op = "./internal/adapters/repository/objectStorage/init.go.DeleteObject()"

	_, err := s3Client.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s3Client.bucketName,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s3Client S3Client) GetPresignURL(ctx context.Context, id string) (string, error) {
	const op = "./internal/adapters/repo/objectStorage/init.go.GetPresignURL()"

	presignClient := s3.NewPresignClient(s3Client.client)

	req := s3.GetObjectInput{
		Bucket: &s3Client.bucketName,
		Key:    &id,
	}

	resp, err := presignClient.PresignGetObject(ctx, &req, s3.WithPresignExpires(expired))
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return resp.URL, nil
}
