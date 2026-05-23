// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	pkgstorage "github.com/choysum-dev/choysum/pkg/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type storedContentDriver struct {
	client        *minio.Client
	defaultBucket string
}

type objectLocator struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

// NewStoredContentDriver builds the S3-backed stored content driver.
func NewStoredContentDriver(att *config.AttachmentConfig) (pkgstorage.StoredContentDriver, error) {
	client, bucket, err := newS3Client(att)
	if err != nil {
		return nil, err
	}
	return &storedContentDriver{client: client, defaultBucket: bucket}, nil
}

func (d *storedContentDriver) Provider() string {
	return "s3"
}

func (d *storedContentDriver) Put(ctx context.Context, input pkgstorage.PutPayloadInput) (pkgstorage.PayloadMutation, error) {
	if d == nil || d.client == nil {
		return pkgstorage.PayloadMutation{}, fmt.Errorf("s3 driver is not initialized")
	}

	key := buildUploadStagingKey(input.UploadID, input.ChecksumSHA256)
	if err := d.putObject(ctx, d.defaultBucket, key, input.Body, input.ContentType); err != nil {
		return pkgstorage.PayloadMutation{}, err
	}

	locatorJSON, err := json.Marshal(objectLocator{Bucket: d.defaultBucket, Key: key})
	if err != nil {
		return pkgstorage.PayloadMutation{}, fmt.Errorf("marshal s3 locator json: %w", err)
	}

	return pkgstorage.PayloadMutation{LocatorJSON: string(locatorJSON)}, nil
}

func (d *storedContentDriver) Open(ctx context.Context, record pkgstorage.StoredContentRecord) ([]byte, error) {
	if d == nil || d.client == nil {
		return nil, fmt.Errorf("s3 driver is not initialized")
	}

	locator, err := parseLocatorJSON(record.LocatorJSON)
	if err != nil {
		return nil, err
	}
	return d.getObject(ctx, locator.Bucket, locator.Key)
}

func (d *storedContentDriver) Delete(ctx context.Context, record pkgstorage.StoredContentRecord) error {
	if d == nil || d.client == nil {
		return fmt.Errorf("s3 driver is not initialized")
	}

	locator, err := parseLocatorJSON(record.LocatorJSON)
	if err != nil {
		return err
	}
	return d.removeObject(ctx, locator.Bucket, locator.Key)
}

func newS3Client(att *config.AttachmentConfig) (*minio.Client, string, error) {
	if att == nil || att.S3 == nil {
		return nil, "", fmt.Errorf("document.attachment.s3 config is required")
	}
	s3 := att.S3
	endpoint := strings.TrimSpace(s3.Endpoint)
	if endpoint == "" {
		return nil, "", fmt.Errorf("document.attachment.s3.endpoint is required")
	}
	bucket := strings.TrimSpace(s3.Bucket)
	if bucket == "" {
		return nil, "", fmt.Errorf("document.attachment.s3.bucket is required")
	}
	accessKey := strings.TrimSpace(s3.AccessKey)
	secretKey := strings.TrimSpace(s3.SecretKey)
	if accessKey == "" || secretKey == "" {
		return nil, "", fmt.Errorf("document.attachment.s3 access key and secret key are required")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: s3.UseTLS,
		Region: strings.TrimSpace(s3.Region),
	})
	if err != nil {
		return nil, "", fmt.Errorf("create s3 client: %w", err)
	}

	return client, bucket, nil
}

func (d *storedContentDriver) putObject(ctx context.Context, bucket, key string, body []byte, contentType string) error {
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	if bucket == "" || key == "" {
		return fmt.Errorf("bucket and key are required for s3 put")
	}

	putOptions := minio.PutObjectOptions{}
	if ct := strings.TrimSpace(contentType); ct != "" {
		putOptions.ContentType = ct
	}

	reader := bytes.NewReader(body)
	if _, err := d.client.PutObject(ctx, bucket, key, reader, int64(len(body)), putOptions); err != nil {
		return fmt.Errorf("put s3 object %s/%s: %w", bucket, key, err)
	}
	return nil
}

func (d *storedContentDriver) getObject(ctx context.Context, bucket, key string) ([]byte, error) {
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	if bucket == "" || key == "" {
		return nil, fmt.Errorf("bucket and key are required for s3 get")
	}

	object, err := d.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get s3 object %s/%s: %w", bucket, key, err)
	}
	defer object.Close()

	payload, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("read s3 object %s/%s: %w", bucket, key, err)
	}
	return payload, nil
}

func (d *storedContentDriver) removeObject(ctx context.Context, bucket, key string) error {
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	if bucket == "" || key == "" {
		return fmt.Errorf("bucket and key are required for s3 delete")
	}

	if err := d.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove s3 object %s/%s: %w", bucket, key, err)
	}
	return nil
}

func parseLocatorJSON(locatorJSON string) (objectLocator, error) {
	text := strings.TrimSpace(locatorJSON)
	if text == "" {
		return objectLocator{}, fmt.Errorf("stored content locator json is required for s3")
	}

	var locator objectLocator
	if err := json.Unmarshal([]byte(text), &locator); err != nil {
		return objectLocator{}, fmt.Errorf("parse s3 locator json: %w", err)
	}
	locator.Bucket = strings.TrimSpace(locator.Bucket)
	locator.Key = strings.TrimSpace(locator.Key)
	if locator.Bucket == "" || locator.Key == "" {
		return objectLocator{}, fmt.Errorf("s3 locator json requires bucket and key")
	}
	return locator, nil
}

func buildUploadStagingKey(uploadID string, checksum string) string {
	id := strings.TrimSpace(uploadID)
	if id == "" {
		id = "upload"
	}

	normalizedChecksum := strings.ToLower(strings.TrimSpace(checksum))
	if len(normalizedChecksum) > 12 {
		normalizedChecksum = normalizedChecksum[:12]
	}
	if normalizedChecksum == "" {
		return fmt.Sprintf("staging/%s", id)
	}

	return fmt.Sprintf("staging/%s/%s", id, normalizedChecksum)
}
