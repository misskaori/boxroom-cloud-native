package store_agent

import (
	"github.com/sirupsen/logrus"
	k8sagent "github.io/misskaori/boxroom-crd/kubernetes/kubernetes/k8s-agent"
	"github.io/misskaori/boxroom-crd/kubernetes/resource/tree"
	"github.io/misskaori/boxroom-crd/kubernetes/storage/dir"
	storeclient "github.io/misskaori/boxroom-crd/kubernetes/storage/store-client"
	utilfunc "github.io/misskaori/boxroom-crd/kubernetes/util/util-func"
	utillog "github.io/misskaori/boxroom-crd/kubernetes/util/util-log"
	"os"
)

type AssistLogStoreAgent struct {
	Client        storeclient.StoreClient
	DirDefinition dir.StorageDirDefinition
	workDir       string
	logLocalDir   string
	logFile       *os.File
	fileLogger    *logrus.Logger

	StatusLogger   tree.Status
	statusLocalDir string
}

func (agent *AssistLogStoreAgent) InitLoggerAgent(root *tree.KubernetesRoot) error {
	localWorkDir, loggerLocalDir, statusLoggerLocalDir := agent.DirDefinition.GetAssistLogLocalDir(root)
	fileLogger, logFile, err := new(utillog.NewLog).GetFileLogger(loggerLocalDir)

	if err != nil {
		return err
	}

	agent.workDir = localWorkDir
	agent.logLocalDir = loggerLocalDir
	agent.logFile = logFile
	agent.fileLogger = fileLogger
	agent.statusLocalDir = statusLoggerLocalDir
	agent.StatusLogger = &tree.MissionStatus{
		MissionKind:   root.TreeKind,
		Status:        k8sagent.StatusSuccess,
		FailedObjects: map[string]error{},
	}

	return nil
}

func (agent *AssistLogStoreAgent) GetLogger() (*logrus.Logger, *os.File, string) {
	return agent.fileLogger, agent.logFile, agent.workDir
}

func (agent *AssistLogStoreAgent) UploadLocalLogger(root *tree.KubernetesRoot) error {
	fileOperator := utilfunc.NewWorkDirFileOperator()
	statusLoggerLocalFile, err := fileOperator.CreateFile(agent.statusLocalDir)
	if err != nil {
		return err
	}

	statusLoggerJson, err := agent.StatusLogger.CovertStructToJson()
	if err != nil {
		return err
	}

	err = fileOperator.WriteFile(statusLoggerLocalFile, statusLoggerJson)
	if err != nil {
		return err
	}

	err = statusLoggerLocalFile.Close()
	if err != nil {
		return err
	}

	remoteLoggerDir, remoteStatusLoggerDir := agent.DirDefinition.GetAssistLogRemoteDir(root)

	localZipLogDir, localZipStatusLogDir, err := agent.DirDefinition.GetAssistLogLocalZipDir(agent.logLocalDir, agent.statusLocalDir)

	if err != nil {
		return err
	}

	localZipLogFile, err := fileOperator.OpenFile(localZipLogDir)
	if err != nil {
		return err
	}

	localZipStatusLogFile, err := fileOperator.OpenFile(localZipStatusLogDir)
	if err != nil {
		return err
	}

	err = agent.Client.UploadObject(remoteLoggerDir, localZipLogFile)
	if err != nil {
		return err
	}

	err = agent.Client.UploadObject(remoteStatusLoggerDir, localZipStatusLogFile)

	return nil
}
