// pkg/storage/s3.go
package storage

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

func NewS3Client() (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg), nil
}

func UploadPDFToS3(
	client *s3.Client,
	bucket string,
	key string,
	pdfBytes []byte,
) error {

	_, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        bytes.NewReader(pdfBytes),
		ContentType: aws.String("application/pdf"),
	})

	return err
}

func CheckFileExistsInS3(client *s3.Client, bucket, key string) (bool, error) {
	_, err := client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		// In AWS SDK v2, we check if the error is a 404 (Not Found)
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func GeneratePresignedURL(
	client *s3.Client,
	bucket string,
	key string,
	expiry time.Duration,
) (string, error) {

	presigner := s3.NewPresignClient(client)

	req, err := presigner.PresignGetObject(context.TODO(),
		&s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		},
		s3.WithPresignExpires(expiry),
	)

	if err != nil {
		return "", err
	}

	return req.URL, nil
}
