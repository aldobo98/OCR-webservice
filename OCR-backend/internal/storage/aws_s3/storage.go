package aws_s3

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type AWS_S3 struct {
	client        *s3.Client
	bucket        string
	logger        *slog.Logger
	presignClient *s3.PresignClient
	region        string
}

const VARIABLE_SET_FORMAT = "%v environment variable is set, using its value, %v"

func NewAWS_S3(logger *slog.Logger, endpoint string, bucket string, region string) *AWS_S3 {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		logger.Error("Either AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY environment variable is empty, S3 cannot initialize, please specify both.")
		return nil
	}
	s3_endpoint := endpoint
	if os.Getenv("AWS_ENDPOINT_URL") != "" {
		logger.Info(fmt.Sprintf(VARIABLE_SET_FORMAT, "AWS_ENDPOINT_URL", os.Getenv("AWS_ENDPOINT_URL")))
		s3_endpoint = os.Getenv("AWS_ENDPOINT_URL")
	}
	if s3_endpoint == "" {
		logger.Error("S3 endpoint must be specified, please specify endpoint or set environment variable AWS_ENDPOINT_URL.")
	}
	if region == "" {
		logger.Info("Region is not passed, setting default value us-east-1")
		region = "us-east-1"
	}
	//A régió csak az S3 api kompatibilitás miatt van, a SeaweedFS nem támogatja a régiókat.
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(s3_endpoint)
		options.UsePathStyle = true
	})
	presignClient := s3.NewPresignClient(client)
	return &AWS_S3{
		client:        client,
		bucket:        bucket,
		logger:        logger,
		region:        region,
		presignClient: presignClient,
	}
}

func (storage *AWS_S3) EnsureBucket() error {
	//HeadBucket -> arra való, hogy megnézzem, létezik-e a bucket
	_, err := storage.client.HeadBucket(context.Background(), &s3.HeadBucketInput{Bucket: aws.String(storage.bucket)})
	//Ha nincs error, akkor létezik a bucket
	if err == nil {
		storage.logger.Info("Bucket already exists")
		return err
	}
	//Ha van error, akkor létre kell hozni a bucketet
	//CreateBucket -> létrehozza a bucketet
	storage.logger.Info("Bucket doesn't exist, creating it now")
	_, err = storage.client.CreateBucket(context.Background(), &s3.CreateBucketInput{Bucket: &storage.bucket})
	//Akár létrejön a bucket, akár nem, a hívó majd eldönti, hogy mit csinál vele
	return err
}

func (storage *AWS_S3) Get_Upload_URL(duration int64, objectKey string) (*s3.PresignedPostRequest, error) {
	request, err := storage.presignClient.PresignPostObject(context.Background(), &s3.PutObjectInput{Bucket: aws.String(storage.bucket), Key: aws.String(objectKey)},
		func(options *s3.PresignPostOptions) { options.Expires = time.Duration(duration) * time.Second })
	if err != nil {
		storage.logger.Error(fmt.Sprintf("Couldn't get a presigned post request to put %v:%v. Here's why: %v\n", storage.bucket, objectKey, err))
		return nil, err
	}
	return request, nil
}

func (storage *AWS_S3) Get_Download_URL(duration int64, objectKey string) (*v4.PresignedHTTPRequest, error) {
	request, err := storage.presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(duration) * time.Second
	})
	if err != nil {
		storage.logger.Error(fmt.Sprintf("Couldn't get a presigned request to get %v:%v. Here's why: %v\n",
			storage.bucket, objectKey, err))
		return nil, err
	}
	return request, nil
}

func (storage *AWS_S3) ObjectExists(ctx context.Context, objectKey string) (bool, error) {
	_, err := storage.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(storage.bucket), Key: aws.String(objectKey)})
	if err != nil {
		return false, err
	}
	return true, nil
}
