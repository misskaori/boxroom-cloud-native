package mac

import (
	k8sagent "boxroom/kubernetes/k8s-agent"
	"boxroom/resource/tree"
)

func GetK8sAgent() (tree.Agent, error) {
	agent, err := (&k8sagent.ApiServerConfig{
		Url:              "https://10.211.55.3",
		Port:             "6443",
		KubernetesConfig: "/etc/kubernetes/admin.conf",
	}).AgentInit()
	if err != nil {
		return nil, err
	}
	return agent, nil
}
