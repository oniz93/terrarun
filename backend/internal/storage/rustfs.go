package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type RustFS struct {
	client *s3.Client
	bucket string
}

func NewRustFS(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*RustFS, error) {
	scheme := "http"
	if useSSL {
		scheme = "https"
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		awsconfig.WithRegion("us-east-1"),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("%s://%s", scheme, endpoint))
		o.UsePathStyle = true
	})

	ctx := context.Background()
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
	}

	return &RustFS{client: client, bucket: bucket}, nil
}

func (s *RustFS) Upload(ctx context.Context, bucket, key string, data []byte) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return "", fmt.Errorf("upload to rustfs: %w", err)
	}
	ep := s.client.Options().BaseEndpoint
	if ep != nil {
		return fmt.Sprintf("https://%s/%s/%s", *ep, s.bucket, key), nil
	}
	return fmt.Sprintf("/%s/%s", s.bucket, key), nil
}

func (s *RustFS) Download(ctx context.Context, bucket, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("download from rustfs: %w", err)
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (s *RustFS) Delete(ctx context.Context, bucket, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}
