package awss3

import (
	storeclient "github.io/misskaori/boxroom-crd/kubernetes/storage/store-client"
	util "github.io/misskaori/boxroom-crd/kubernetes/util/util-log"
	"net/rpc"
)

var log = new(util.NewLog).GetLogger()

type S3Config struct {
	StoragePluginUrl string
}

func (config *S3Config) ClientInit() (storeclient.StoreClient, error) {
	rpcClient, err := rpc.DialHTTP("tcp", config.StoragePluginUrl)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	storeClient := &BoxroomStoreClient{
		RpcClient: rpcClient,
	}

	return storeClient, nil
}
