package store_agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"github.com/sirupsen/logrus"
	"github.io/misskaori/boxroom-crd/kubernetes/resource/immobile"
	"github.io/misskaori/boxroom-crd/kubernetes/resource/tree"
	"github.io/misskaori/boxroom-crd/kubernetes/storage/dir"
	storeclient "github.io/misskaori/boxroom-crd/kubernetes/storage/store-client"
	globleimmobile "github.io/misskaori/boxroom-crd/kubernetes/util/globle-immobile"
	utilfunc "github.io/misskaori/boxroom-crd/kubernetes/util/util-func"
	utillog "github.io/misskaori/boxroom-crd/kubernetes/util/util-log"
)

var log = new(utillog.NewLog).GetLogger()

type CoreStoreAgent struct {
	Client        storeclient.StoreClient
	DirDefinition dir.StorageDirDefinition
}

func (agent *CoreStoreAgent) GetResourceTree(root *tree.KubernetesRoot, filters map[string]tree.Filter, ctx context.Context) (*tree.KubernetesRoot, error) {

	fileLoggerValue := ctx.Value(globleimmobile.FileLogger)
	if fileLoggerValue == nil {
		return nil, errors.New("there is no file logger which is request when get resource tree")
	}
	fileLogger := fileLoggerValue.(*logrus.Logger)

	root.TreeKind = immobile.TreeBackupKind

	exist, err := agent.CheckRemoteStorageExist(root)
	if err != nil {
		fileLogger.Error(err)
		return nil, err
	}

	if !exist {
		e := fmt.Sprintf("there is no such %s for cluster %s, %s name is: %s", root.TreeKind, root.Name, root.TreeKind, root.TreeName)
		fileLogger.Error(e)
		return nil, errors.New(e)
	}

	fileOperator := utilfunc.NewWorkDirFileOperator()

	downloadDirMap, workDir, localStorageDir, err := agent.DirDefinition.GetDownLoadDirMap(root)

	for remoteFile, localFile := range downloadDirMap {
		err = agent.GetRemoteStorage(remoteFile, localFile)
		if err != nil {
			fileLogger.Error(err)
			return nil, err
		}
		err = agent.DirDefinition.TransformLocalToResourceTree(localStorageDir, localFile, root)
		if err != nil {
			fileLogger.Error(err)
		}
		break
	}

	defer func() {
		err = fileOperator.DeleteDirOrFile(workDir)
		if err != nil {
			fileLogger.Error(err)
		}
	}()

	filtrateResources(root, filters)

	return root, nil
}

func (agent *CoreStoreAgent) ApplyResourceTree(root *tree.KubernetesRoot, ctx context.Context) error {

	fileLoggerValue := ctx.Value(globleimmobile.FileLogger)
	if fileLoggerValue == nil {
		return errors.New("there is no file logger which is request when get resource tree")
	}
	fileLogger := fileLoggerValue.(*logrus.Logger)

	agent.DirDefinition.PreHandleRestoreRoot(root)

	if root.Groups == nil || len(root.Groups) == 0 {
		return nil
	}

	exist, err := agent.CheckRemoteStorageExist(root)
	if err != nil {
		fileLogger.Error(err)
		return err
	}
	if exist {
		e := fmt.Sprintf("the storage has already exist: storage kind: %s storage name: %s", root.TreeKind, root.TreeName)
		fileLogger.Error(e)
		return errors.New(e)
	}

	fileOperator := utilfunc.NewWorkDirFileOperator()

	storageMap, localWorkDir, err := agent.DirDefinition.TransformResourceTreeToLocal(root)
	defer func() {
		err = fileOperator.DeleteDirOrFile(localWorkDir)
		if err != nil {
			fileLogger.Error(err)
		}
	}()

	if err != nil {
		fileLogger.Error(err)
		return err
	}

	for localFile, remoteFile := range storageMap {
		fileReader, err := fileOperator.OpenFile(localFile)
		if err != nil {
			fileLogger.Error(err)
			return err
		}
		err = agent.Client.UploadObject(remoteFile, fileReader)
		if err != nil {
			fileLogger.Error(err)
			return err
		}
	}

	return nil
}

func (agent *CoreStoreAgent) CheckRemoteStorageExist(root *tree.KubernetesRoot) (bool, error) {
	set, err := agent.ListRemoteStorage(root)
	if err != nil {
		log.Error(err)
		return false, err
	}

	if set.Contains(root.TreeName) {
		return true, nil
	}

	return false, nil
}

func (agent *CoreStoreAgent) ListRemoteStorage(root *tree.KubernetesRoot) (mapset.Set, error) {
	prefix, delimiter := agent.DirDefinition.GetRemoteStoragePrefixAndDelimiter(root)

	list, err := agent.Client.ListCommonPrefix(prefix, delimiter)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	set := mapset.NewSet()
	for _, prefixDir := range list {
		set.Add(agent.DirDefinition.ParseCommonPrefix(prefixDir))
	}

	return set, nil
}

func (agent *CoreStoreAgent) GetRemoteStorage(remoteFile, localFile string) error {
	fileOperator := utilfunc.NewWorkDirFileOperator()

	remoteFileReader, err := agent.Client.GetObject(remoteFile)
	if err != nil {
		log.Error(err)
		return err
	}

	buff := bytes.Buffer{}
	_, err = buff.ReadFrom(remoteFileReader)
	if err != nil {
		log.Error(err)
		return err
	}

	localFileReader, err := fileOperator.CreateFile(localFile)

	err = fileOperator.WriteFile(localFileReader, buff.Bytes())
	if err != nil {
		log.Error(err)
		return err
	}

	defer func() {
		err = localFileReader.Close()
		if err != nil {
			log.Error(err)
		}
	}()

	return nil
}

func filtrateResources(root *tree.KubernetesRoot, filters map[string]tree.Filter) {
	for _, filter := range filters {
		deepFiltrateResources(root, filter)
	}
}

func deepFiltrateResources(resources tree.Resources, filter tree.Filter) {
	if resources.GetKind() != filter.GetFilterKind() {
		for _, children := range resources.ListChildren() {
			deepFiltrateResources(resources.GetChildren(children), filter)
		}
		if len(resources.ListChildren()) == 0 {
			resources.GetParent().DeleteChildren(resources)
		}
		return
	}

	addFlag := (filter.GetFilterPattern() && filter.GetFilterSet().Contains(resources.GetName())) || (!filter.GetFilterPattern() && !filter.GetFilterSet().Contains(resources.GetName()))

	if !addFlag {
		log.Infof("delete resources from resource tree: filter: %v kind: %s name: %s", filter, resources.GetKind(), resources.GetName())
		resources.GetParent().DeleteChildren(resources)
	}

}
