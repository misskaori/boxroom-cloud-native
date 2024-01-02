package windows

import (
	"boxroom/api/parameter"
	k8sagent "boxroom/kubernetes/k8s-agent"
	"boxroom/resource/immobile"
	"encoding/json"
	"fmt"
	"testing"
)

func TestStorageAgent(t *testing.T) {

}

func TestRestfulApi(t *testing.T) {
	input := parameter.StoreInputParameter{
		Kind:     immobile.RootKind,
		Name:     immobile.RootName,
		TreeKind: immobile.TreeBackupKind,
		TreeName: "test-backup-crd-20230417212",
		Filters:  map[string]parameter.FilterInputParameter{},
		KubernetesConfig: &parameter.KubernetesConfigInputParameter{
			Url:                 "https://192.168.100.140",
			Port:                "6443",
			ConfigFile:          "/etc/kubernetes/admin.conf",
			ServiceAccountToken: "",
		},
		AccessType: k8sagent.KubeConfigFileType,
	}

	input.Filters[immobile.NamespaceKind] = parameter.FilterInputParameter{
		Kind:              immobile.NamespaceKind,
		ResourceInclude:   false,
		ResourceFilterSet: []string{"velero"},
	}

	jsons, _ := json.Marshal(input)
	str := string(jsons)
	fmt.Println(str)
}
