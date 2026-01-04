package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type s3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// Uploader uploads episode artifacts to S3.
type Uploader struct {
	client s3API
	bucket string
	prefix string
}

func New(ctx context.Context, bucket, prefix, region string) (*Uploader, error) {
	if bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	if region == "" {
		region = "us-west-2"
	}
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg)
	return &Uploader{
		client: client,
		bucket: bucket,
		prefix: normalizePrefix(prefix),
	}, nil
}

func NewWithClient(bucket, prefix string, client s3API) *Uploader {
	return &Uploader{
		client: client,
		bucket: bucket,
		prefix: normalizePrefix(prefix),
	}
}

func (u *Uploader) Bucket() string { return u.bucket }
func (u *Uploader) Prefix() string { return u.prefix }

func (u *Uploader) KeyForDate(t time.Time, filename string) string {
	y, m, d := t.UTC().Date()
	return joinKey(u.prefix, fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", int(m)), fmt.Sprintf("%02d", d), filename)
}

func (u *Uploader) KeyForLatest(filename string) string {
	return joinKey(u.prefix, "latest", filename)
}

// UploadFile uploads a local file to the given key.
func (u *Uploader) UploadFile(ctx context.Context, key, localPath, contentType, cacheControl string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	input := &s3.PutObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   f,
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if cacheControl != "" {
		input.CacheControl = aws.String(cacheControl)
	}
	_, err = u.client.PutObject(ctx, input)
	return err
}

// UploadBytes uploads in-memory data to the given key.
func (u *Uploader) UploadBytes(ctx context.Context, key string, data []byte, contentType, cacheControl string) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if cacheControl != "" {
		input.CacheControl = aws.String(cacheControl)
	}
	_, err := u.client.PutObject(ctx, input)
	return err
}

// DownloadBytes downloads an object into memory.
func (u *Uploader) DownloadBytes(ctx context.Context, key string) ([]byte, error) {
	out, err := u.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

// CopyToLatest copies an existing object to the latest key.
func (u *Uploader) CopyToLatest(ctx context.Context, srcKey, filename, contentType, cacheControl string) error {
	latestKey := u.KeyForLatest(filename)
	copySource := encodeCopySource(u.bucket, srcKey)
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(u.bucket),
		Key:        aws.String(latestKey),
		CopySource: aws.String(copySource),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if cacheControl != "" {
		input.CacheControl = aws.String(cacheControl)
	}
	if contentType != "" || cacheControl != "" {
		input.MetadataDirective = types.MetadataDirectiveReplace
	}
	_, err := u.client.CopyObject(ctx, input)
	return err
}

func normalizePrefix(prefix string) string {
	return strings.Trim(prefix, "/")
}

func joinKey(prefix string, parts ...string) string {
	all := []string{}
	if prefix != "" {
		all = append(all, prefix)
	}
	all = append(all, parts...)
	key := path.Join(all...)
	return strings.TrimPrefix(key, "/")
}

func encodeCopySource(bucket, key string) string {
	parts := strings.Split(key, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return bucket + "/" + strings.Join(parts, "/")
}

// IsNotFound returns true when the error indicates the object does not exist.
func IsNotFound(err error) bool {
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "NoSuchKey" || code == "NotFound"
	}
	return false
}
