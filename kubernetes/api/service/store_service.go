package service

import (
	"github.io/misskaori/boxroom-crd/kubernetes/controller"
	"github.io/misskaori/boxroom-crd/kubernetes/resource/tree"
	utillog "github.io/misskaori/boxroom-crd/kubernetes/util/util-log"
)

func BackupService(agentController *controller.AgentController, root *tree.KubernetesRoot, filters map[string]tree.Filter) error {
	err := agentController.Backup(root, filters)
	if err != nil {
		utillog.Logger.Error(err)
		return err
	}

	return nil
}

func RestoreService(agentController *controller.AgentController, root *tree.KubernetesRoot, filters map[string]tree.Filter) error {
	err := agentController.Restore(root, filters)
	if err != nil {
		utillog.Logger.Error(err)
		return err
	}

	return nil
}
