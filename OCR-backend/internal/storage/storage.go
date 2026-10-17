package storage

import (
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Storage interface {
	EnsureBucket() error
	Get_Upload_URL(duration_seconds int64, objectKey string) (*s3.PresignedPostRequest, error)
	Get_Download_URL(duration_seconds int64, objectKey string) (*v4.PresignedHTTPRequest, error)
}
