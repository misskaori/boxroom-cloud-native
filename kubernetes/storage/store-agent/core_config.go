package store_agent

import (
	"errors"
	"fmt"
	"github.io/misskaori/boxroom-crd/kubernetes/resource/tree"
	"github.io/misskaori/boxroom-crd/kubernetes/storage/dir"
	storeclient "github.io/misskaori/boxroom-crd/kubernetes/storage/store-client"
)

type StorageConfig struct {
	Client        storeclient.StoreClient
	DirDefinition dir.StorageDirDefinition
}

func (config *StorageConfig) AgentInit() (tree.Agent, error) {
	if config.Client == nil || config.DirDefinition == nil {
		e := fmt.Sprintf("config do not contains enough parameter")
		log.Error(e)
		return nil, errors.New(e)
	}
	agent := &CoreStoreAgent{
		Client:        config.Client,
		DirDefinition: config.DirDefinition,
	}
	return agent, nil
}
