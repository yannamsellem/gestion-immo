package r2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

var (
	// ErrObjectNotFound indicates the requested R2 object key does not exist.
	ErrObjectNotFound = errors.New("r2: object not found")
	// ErrInvalidKey indicates the provided R2 object key is empty or invalid.
	ErrInvalidKey = errors.New("r2: object key is required")
)

// ObjectStorage defines the Cloudflare R2 (S3-compatible) object storage contract.
type ObjectStorage interface {
	Upload(ctx context.Context, key string, contentType string, body io.Reader) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	PresignGetURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

// S3Storage implements ObjectStorage against a Cloudflare R2 / S3-compatible endpoint
// using the official github.com/aws/aws-sdk-go-v2/service/s3 SDK.
type S3Storage struct {
	endpoint        string
	accessKeyID     string
	secretAccessKey string
	bucketName      string
	region          string
	httpClient      *http.Client
	s3Client        *s3.Client
	presignClient   *s3.PresignClient
}

// NewS3Storage constructs a new S3Storage client for Cloudflare R2 backed by aws-sdk-go-v2.
func NewS3Storage(endpoint, accessKeyID, secretAccessKey, bucketName string) *S3Storage {
	s := &S3Storage{
		endpoint:        strings.TrimRight(strings.TrimSpace(endpoint), "/"),
		accessKeyID:     strings.TrimSpace(accessKeyID),
		secretAccessKey: strings.TrimSpace(secretAccessKey),
		bucketName:      strings.Trim(strings.TrimSpace(bucketName), "/"),
		region:          "auto",
		httpClient:      &http.Client{Timeout: 30 * time.Second},
	}
	s.rebuildClients()
	return s
}

// WithHTTPClient returns a copy of S3Storage configured with the given *http.Client.
func (s *S3Storage) WithHTTPClient(client *http.Client) *S3Storage {
	if client == nil {
		return s
	}
	copyS := *s
	copyS.httpClient = client
	copyS.rebuildClients()
	return &copyS
}

func (s *S3Storage) rebuildClients() {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(s.region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s.accessKeyID, s.secretAccessKey, ""),
		),
		config.WithHTTPClient(s.httpClient),
	)
	if err != nil {
		cfg = aws.Config{
			Region:      s.region,
			Credentials: credentials.NewStaticCredentialsProvider(s.accessKeyID, s.secretAccessKey, ""),
			HTTPClient:  s.httpClient,
		}
	}

	s.s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		if s.endpoint != "" {
			o.BaseEndpoint = aws.String(s.endpoint)
		}
		o.UsePathStyle = true
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})
	s.presignClient = s3.NewPresignClient(s.s3Client)
}

func (s *S3Storage) validateEndpoint() error {
	parsed, err := url.Parse(s.endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("r2: invalid endpoint %q", s.endpoint)
	}
	return nil
}

// Upload uploads an object to Cloudflare R2 using s3.Client.PutObject.
func (s *S3Storage) Upload(ctx context.Context, key string, contentType string, body io.Reader) error {
	cleanKey, err := normalizeKey(key)
	if err != nil {
		return err
	}
	if body == nil {
		return fmt.Errorf("r2: upload body must not be nil")
	}
	if err := s.validateEndpoint(); err != nil {
		return err
	}

	payload, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("r2: read upload body: %w", err)
	}

	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}

	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucketName),
		Key:           aws.String(cleanKey),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(len(payload))),
		Body:          bytes.NewReader(payload),
	})
	if err != nil {
		return fmt.Errorf("r2: PutObject %s: %w", cleanKey, err)
	}
	return nil
}

// Download retrieves an object stream from Cloudflare R2 using s3.Client.GetObject.
func (s *S3Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	cleanKey, err := normalizeKey(key)
	if err != nil {
		return nil, err
	}
	if err := s.validateEndpoint(); err != nil {
		return nil, err
	}

	out, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(cleanKey),
	})
	if err != nil {
		if isS3NotFoundError(err) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("r2: GetObject %s: %w", cleanKey, err)
	}

	return out.Body, nil
}

// PresignGetURL generates a pre-signed GET URL for key valid for ttl using s3.PresignClient.
func (s *S3Storage) PresignGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	cleanKey, err := normalizeKey(key)
	if err != nil {
		return "", err
	}
	if err := s.validateEndpoint(); err != nil {
		return "", err
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	presigned, err := s.presignClient.PresignGetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(cleanKey),
		},
		s3.WithPresignExpires(ttl),
	)
	if err != nil {
		return "", fmt.Errorf("r2: PresignGetObject %s: %w", cleanKey, err)
	}

	return presigned.URL, nil
}

func isS3NotFoundError(err error) bool {
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		if code == "NoSuchKey" || code == "NotFound" || code == "404" {
			return true
		}
	}
	var respErr interface{ HTTPStatusCode() int }
	if errors.As(err, &respErr) && respErr.HTTPStatusCode() == http.StatusNotFound {
		return true
	}
	return false
}

func normalizeKey(key string) (string, error) {
	trimmed := strings.TrimSpace(strings.TrimLeft(key, "/"))
	if trimmed == "" || strings.Contains(trimmed, "..") {
		return "", ErrInvalidKey
	}
	return trimmed, nil
}

type storedObject struct {
	contentType string
	data        []byte
}

// InMemoryStorage is a thread-safe in-memory implementation of ObjectStorage for unit and integration tests.
type InMemoryStorage struct {
	mu      sync.RWMutex
	objects map[string]storedObject
	baseURL string
}

// NewInMemoryStorage constructs a new empty thread-safe InMemoryStorage.
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		objects: make(map[string]storedObject),
		baseURL: "https://r2.internal.test",
	}
}

// Upload stores the object bytes and contentType in memory under key.
func (m *InMemoryStorage) Upload(_ context.Context, key string, contentType string, body io.Reader) error {
	cleanKey, err := normalizeKey(key)
	if err != nil {
		return err
	}
	if body == nil {
		return fmt.Errorf("r2: upload body must not be nil")
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("r2: read upload body: %w", err)
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[cleanKey] = storedObject{
		contentType: contentType,
		data:        append([]byte(nil), data...),
	}
	return nil
}

// Download returns an io.ReadCloser over the stored bytes for key.
func (m *InMemoryStorage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	cleanKey, err := normalizeKey(key)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	obj, exists := m.objects[cleanKey]
	m.mu.RUnlock()
	if !exists {
		return nil, ErrObjectNotFound
	}

	return io.NopCloser(bytes.NewReader(append([]byte(nil), obj.data...))), nil
}

// PresignGetURL returns a deterministic signed URL for key with the requested TTL in seconds.
func (m *InMemoryStorage) PresignGetURL(_ context.Context, key string, ttl time.Duration) (string, error) {
	cleanKey, err := normalizeKey(key)
	if err != nil {
		return "", err
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return fmt.Sprintf("%s/%s?X-Amz-Expires=%d", m.baseURL, cleanKey, int64(ttl.Seconds())), nil
}
