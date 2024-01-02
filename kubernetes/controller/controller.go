package controller

import (
	"context"
	"github.io/misskaori/boxroom-crd/kubernetes/resource/immobile"
	"github.io/misskaori/boxroom-crd/kubernetes/resource/tree"
	"github.io/misskaori/boxroom-crd/kubernetes/storage/dir"
	storeagent "github.io/misskaori/boxroom-crd/kubernetes/storage/store-agent"
	storeclient "github.io/misskaori/boxroom-crd/kubernetes/storage/store-client"
	globleimmobile "github.io/misskaori/boxroom-crd/kubernetes/util/globle-immobile"
	utilfunc "github.io/misskaori/boxroom-crd/kubernetes/util/util-func"
	utillog "github.io/misskaori/boxroom-crd/kubernetes/util/util-log"
)

type AgentController struct {
	KubernetesAgent tree.Agent
	StorageClient   storeclient.StoreClient
	DirDefinition   dir.StorageDirDefinition
}

func (controller *AgentController) Backup(root *tree.KubernetesRoot, filters map[string]tree.Filter) error {
	coreStorageAgent, assistStorageAgent, err := getStorageAgent(controller.StorageClient, controller.DirDefinition)
	if err != nil {
		utillog.Logger.Error(err)
		return err
	}

	err = assistStorageAgent.InitLoggerAgent(root)
	fileLogger, logFile, workDir := assistStorageAgent.GetLogger()

	defer func() {
		err = logFile.Close()
		if err != nil {
			utillog.Logger.Error(err)
		}

		fileOperator := utilfunc.NewWorkDirFileOperator()
		err = fileOperator.DeleteDirOrFile(workDir)
		if err != nil {
			utillog.Logger.Error(err)
		}
	}()

	if err != nil {
		utillog.Logger.Error(err)
		return err
	}

	ctx := context.WithValue(context.Background(), globleimmobile.FileLogger, fileLogger)
	ctx = context.WithValue(ctx, globleimmobile.MissionStatus, assistStorageAgent.StatusLogger)

	fileLogger.Info("begin to backup")

	root, err = controller.KubernetesAgent.GetResourceTree(root, filters, ctx)
	if err != nil {
		utillog.Logger.Error(err)
		return err
	}

	err = coreStorageAgent.ApplyResourceTree(root, ctx)
	if err != nil {
		utillog.Logger.Error(err)
		return err
	}

	fileLogger.Info("backup is completed")

	err = assistStorageAgent.UploadLocalLogger(root)
	if err != nil {
		utillog.Logger.Error(err)
		return err
	}

	return nil
}

func (controller *AgentController) Restore(root *tree.KubernetesRoot, filters map[string]tree.Filter) error {
	coreStorageAgent, assistStorageAgent, err := getStorageAgent(controller.StorageClient, controller.DirDefinition)
	if err != nil {
		utillog.Logger.Error(err)
		return err
	}

	err = assistStorageAgent.InitLoggerAgent(root)
	fileLogger, logFile, workDir := assistStorageAgent.GetLogger()

	defer func() {
		err := logFile.Close()
		if err != nil {
			utillog.Logger.Error(err)
		}

		fileOperator := utilfunc.NewWorkDirFileOperator()
		err = fileOperator.DeleteDirOrFile(workDir)
		if err != nil {
			utillog.Logger.Error(err)
		}
	}()

	if err != nil {
		utillog.Logger.Error(err)
		return err
	}

	ctx := context.WithValue(context.Background(), globleimmobile.FileLogger, fileLogger)
	ctx = context.WithValue(ctx, globleimmobile.MissionStatus, assistStorageAgent.StatusLogger)

	fileLogger.Info("begin to restore")

	root, err = coreStorageAgent.GetResourceTree(root, filters, ctx)
	if err != nil {
		fileLogger.Error(err)
		return err
	}

	err = controller.KubernetesAgent.ApplyResourceTree(root, ctx)
	if err != nil {
		fileLogger.Error(err)
		return err
	}

	root.TreeKind = immobile.TreeRestoreKind
	err = coreStorageAgent.ApplyResourceTree(root, ctx)
	if err != nil {
		utillog.Logger.Error(err)
		return err
	}

	fileLogger.Info("restore is completed")

	err = assistStorageAgent.UploadLocalLogger(root)
	if err != nil {
		utillog.Logger.Error(err)
		return err
	}

	return nil
}

func getStorageAgent(storageClient storeclient.StoreClient, dirDefinition dir.StorageDirDefinition) (tree.Agent, *storeagent.AssistLogStoreAgent, error) {
	coreStorageAgent, err := (&storeagent.StorageConfig{
		Client:        storageClient,
		DirDefinition: dirDefinition,
	}).AgentInit()

	if err != nil {
		return nil, nil, err
	}

	assistStorageAgent := &storeagent.AssistLogStoreAgent{
		Client:        storageClient,
		DirDefinition: dirDefinition,
	}

	return coreStorageAgent, assistStorageAgent, err
}
