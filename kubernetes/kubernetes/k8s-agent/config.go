package k8s_agent

import (
	"errors"
	"github.io/misskaori/boxroom-crd/kubernetes/resource/tree"
	utillog "github.io/misskaori/boxroom-crd/kubernetes/util/util-log"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var log = new(utillog.NewLog).GetLogger()

type ApiServerConfig struct {
	Url                 string
	KubernetesConfig    string
	ServiceAccountToken string
	AccessType          string
}

func (config *ApiServerConfig) AgentInit() (tree.Agent, error) {
	apiServerUrl := config.Url

	var configObject *restclient.Config
	var err error

	switch config.AccessType {
	case KubeConfigFileType:
		if len(config.KubernetesConfig) == 0 {
			return nil, errors.New("kubernetes kubeconfig file is empty")
		}
		configObject, err = clientcmd.BuildConfigFromFlags(apiServerUrl, config.KubernetesConfig)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	case ServiceAccountTokenType:
		if len(config.ServiceAccountToken) == 0 {
			return nil, errors.New("kubernetes service account token is empty")
		}
		configObject = &restclient.Config{
			Host:        apiServerUrl,
			BearerToken: config.ServiceAccountToken,
			TLSClientConfig: restclient.TLSClientConfig{
				Insecure: true,
			},
		}
	case InClusterConfigType:
		configObject, err = restclient.InClusterConfig()
		if err != nil {
			log.Error(err)
			return nil, err
		}
	default:
		return nil, errors.New("kubernetes config access type is wrong: " + config.AccessType)
	}

	dynamicClient, err := config.getDynamicClient(configObject)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	discoveryClient, err := config.getDiscoveryClient(configObject)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	clientSet, err := config.getClientSet(configObject)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	client := &KubernetesAgent{
		ConfigObject:    configObject,
		DynamicClient:   dynamicClient,
		DiscoveryClient: discoveryClient,
		ClientSet:       clientSet,
	}

	return client, err
}

func (config *ApiServerConfig) getDynamicClient(configObject *restclient.Config) (*dynamic.DynamicClient, error) {
	dynamicClient, err := dynamic.NewForConfig(configObject)
	return dynamicClient, err
}

func (config *ApiServerConfig) getDiscoveryClient(configObject *restclient.Config) (*discovery.DiscoveryClient, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(configObject)
	return discoveryClient, err
}

func (config *ApiServerConfig) getClientSet(configObject *restclient.Config) (*kubernetes.Clientset, error) {
	clientSetClient, err := kubernetes.NewForConfig(configObject)
	return clientSetClient, err
}
