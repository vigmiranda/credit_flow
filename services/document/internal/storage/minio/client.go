package minio

import (
	"context"
	"fmt"
	"io"
	"strings"

	minioapi "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint       string
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	BucketName     string
	UseSSL         bool
}

type Client struct {
	client         *minioapi.Client
	publicEndpoint string
	bucketName     string
}

func New(config Config) (*Client, error) {
	client, err := minioapi.New(config.Endpoint, &minioapi.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		client:         client,
		publicEndpoint: strings.TrimRight(config.PublicEndpoint, "/"),
		bucketName:     config.BucketName,
	}, nil
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucketName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return c.client.MakeBucket(ctx, c.bucketName, minioapi.MakeBucketOptions{})
}

func (c *Client) Upload(ctx context.Context, fileKey, contentType string, body io.Reader, size int64) error {
	_, err := c.client.PutObject(ctx, c.bucketName, fileKey, body, size, minioapi.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (c *Client) PublicURL(fileKey string) string {
	return fmt.Sprintf("%s/%s/%s", c.publicEndpoint, c.bucketName, strings.TrimLeft(fileKey, "/"))
}
