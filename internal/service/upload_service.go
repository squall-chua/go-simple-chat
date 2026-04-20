package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"go-simple-chat/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

type UploadService struct {
	client     *s3.Client
	bucket     string
	publicHost string
}

func NewUploadService(cfg *config.Config) (*UploadService, error) {
	// 1. Initialize AWS SDK V2 Config
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if cfg.S3Endpoint != "" && (strings.Contains(cfg.S3Endpoint, "localhost") || strings.Contains(cfg.S3Endpoint, "minio")) {
			return aws.Endpoint{
				PartitionID:       "aws",
				URL:               fmt.Sprintf("http://%s", cfg.S3Endpoint),
				SigningRegion:     cfg.S3Region,
				HostnameImmutable: true,
			}, nil
		}
		// Fallback to default resolver
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsCfg, err := s3config.LoadDefaultConfig(context.Background(),
		s3config.WithRegion(cfg.S3Region),
		s3config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")),
		s3config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // Required for MinIO and some S3-compatible providers
	})

	// 2. Ensure bucket exists (helpful for local development)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(cfg.S3Bucket),
	})
	if err != nil {
		// Bucket might not exist, try to create it
		_, createErr := client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(cfg.S3Bucket),
			CreateBucketConfiguration: &types.CreateBucketConfiguration{
				LocationConstraint: types.BucketLocationConstraint(cfg.S3Region),
			},
		})
		if createErr == nil {
			// Set public policy
			policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetBucketLocation","s3:ListBucket"],"Resource":["arn:aws:s3:::%s"]},{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, cfg.S3Bucket, cfg.S3Bucket)
			_, _ = client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
				Bucket: aws.String(cfg.S3Bucket),
				Policy: aws.String(policy),
			})
		}
	}

	// Determine public host for URL generation
	publicHost := cfg.S3PublicURL
	if publicHost == "" {
		host := cfg.S3Endpoint
		if host == "minio:9000" {
			host = "localhost:9000"
		}
		protocol := "http"
		if cfg.S3UseSSL {
			protocol = "https"
		}
		publicHost = fmt.Sprintf("%s://%s/%s", protocol, host, cfg.S3Bucket)
	}

	return &UploadService{
		client:     client,
		bucket:     cfg.S3Bucket,
		publicHost: publicHost,
	}, nil
}

func (s *UploadService) UploadFile(ctx context.Context, fileName string, reader io.Reader, size int64, contentType string) (string, error) {
	uniqueName := fmt.Sprintf("%s%s", uuid.New().String(), filepath.Ext(fileName))

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(uniqueName),
		Body:        reader,
		ContentType: aws.String(contentType),
		// ACL: types.ObjectCannedACLPublicRead, // Some S3 providers don't support ACLs or require bucket config
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Final URL Construction
	if strings.Contains(s.publicHost, s.bucket) {
		return fmt.Sprintf("%s/%s", s.publicHost, uniqueName), nil
	}
	return fmt.Sprintf("%s/%s/%s", s.publicHost, s.bucket, uniqueName), nil
}
