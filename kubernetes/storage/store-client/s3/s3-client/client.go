package awss3

import (
	storeclient "boxroom/storage/store-client"
	"bytes"
	"io"
	"net/rpc"
	"os"
)

type BoxroomStoreClient struct {
	RpcClient *rpc.Client
}

func (client *BoxroomStoreClient) ListBucket() ([]string, error) {
	input := &storeclient.ListBucketInput{}
	output := &storeclient.ListBucketOutput{}
	err := client.RpcClient.Call("S3Client.ListBucket", input, output)
	if err != nil {
		return nil, err
	}
	return output.Buckets, nil
}

func (client *BoxroomStoreClient) ListObjects(prefix string) ([]string, error) {
	input := &storeclient.ListObjectsInput{
		Prefix: prefix,
	}
	output := &storeclient.ListObjectsOutput{}
	err := client.RpcClient.Call("S3Client.ListObjects", input, output)
	if err != nil {
		return nil, err
	}
	return output.Objects, nil
}

func (client *BoxroomStoreClient) ListCommonPrefix(prefix, delimiter string) ([]string, error) {
	input := &storeclient.ListCommonPrefixInput{
		Prefix:    prefix,
		Delimiter: delimiter,
	}
	output := &storeclient.ListCommonPrefixOutput{}
	err := client.RpcClient.Call("S3Client.ListCommonPrefix", input, output)
	if err != nil {
		return nil, err
	}
	return output.TreeName, nil
}

func (client *BoxroomStoreClient) CreateBucket(bucketName string) error {
	input := &storeclient.CreateBucketInput{
		BucketName: bucketName,
	}
	output := &storeclient.CreateBucketOutput{}
	err := client.RpcClient.Call("S3Client.CreateBucket", input, output)
	if err != nil {
		return err
	}
	return nil
}

func (client *BoxroomStoreClient) GetObject(key string) (io.Reader, error) {
	input := &storeclient.GetObjectInput{
		Key: key,
	}
	output := &storeclient.GetObjectOutput{}
	err := client.RpcClient.Call("S3Client.GetObject", input, output)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(output.File)

	return reader, nil
}

func (client *BoxroomStoreClient) UploadObject(fileName string, body *os.File) error {
	buffer := bytes.Buffer{}
	_, err := buffer.ReadFrom(body)
	if err != nil {
		return err
	}

	input := &storeclient.UploadObjectInput{
		FileName: fileName,
		Body:     buffer.Bytes(),
	}
	output := &storeclient.UploadObjectOutput{}
	err = client.RpcClient.Call("S3Client.UploadObject", input, output)
	if err != nil {
		return err
	}
	return nil
}

func (client *BoxroomStoreClient) StoragePluginHealthCheck() error {
	input := &storeclient.StoragePluginHealthCheckInput{
		CheckMethods: "ListBucket",
	}
	output := &storeclient.StoragePluginHealthCheckOutput{}
	err := client.RpcClient.Call("S3Client.StoragePluginHealthCheck", input, output)
	if err != nil {
		return err
	}
	return nil
}
