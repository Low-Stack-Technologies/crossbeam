package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	appconfig "github.com/low-stack-technologies/crossbeam/server/internal/config"
)

type Client struct {
	s3     *s3.Client
	bucket string
}

func New(cfg *appconfig.Config) *Client {
	creds := credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")

	awsCfg, _ := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(creds),
	)

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		o.UsePathStyle = true
	})

	return &Client{s3: s3Client, bucket: cfg.S3Bucket}
}

func (c *Client) UploadFile(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	})
	return err
}

func (c *Client) DeleteFile(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (c *Client) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presigner := s3.NewPresignClient(c.s3)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign get object: %w", err)
	}
	return req.URL, nil
}

func (c *Client) GenerateKey(userID, fileName string) string {
	sanitized := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' {
			return '-'
		}
		return r
	}, fileName)
	return fmt.Sprintf("users/%s/%d-%s", userID, time.Now().UnixMilli(), sanitized)
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	_, err := c.s3.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		var bae *types.BucketAlreadyExists
		var baoby *types.BucketAlreadyOwnedByYou
		if !errors.As(err, &bae) && !errors.As(err, &baoby) {
			return fmt.Errorf("create bucket: %w", err)
		}
	}
	return nil
}
