package store_client

import (
	"io"
	"os"
)

type StoreClient interface {
	ListBucket() ([]string, error)
	ListObjects(prefix string) ([]string, error)
	ListCommonPrefix(prefix, delimiter string) ([]string, error)
	CreateBucket(bucketName string) error
	GetObject(key string) (io.Reader, error)
	UploadObject(fileName string, body *os.File) error
	StoragePluginHealthCheck() error
}

type StoreClientConfig interface {
	ClientInit() (StoreClient, error)
}

type ListBucketInput struct {
}

type ListObjectsInput struct {
	Prefix string
}

type ListCommonPrefixInput struct {
	Prefix    string
	Delimiter string
}

type CreateBucketInput struct {
	BucketName string
}

type GetObjectInput struct {
	Key string
}

type UploadObjectInput struct {
	FileName string
	Body     []byte
}

type StoragePluginHealthCheckInput struct {
	CheckMethods string
}

type ListBucketOutput struct {
	Buckets []string
}

type ListObjectsOutput struct {
	Objects []string
}

type ListCommonPrefixOutput struct {
	TreeName []string
}

type CreateBucketOutput struct {
	CreateBucketStatus bool
}

type GetObjectOutput struct {
	File []byte
}

type UploadObjectOutput struct {
	UploadObjectStatus bool
}

type StoragePluginHealthCheckOutput struct {
	HealthStatus bool
}
