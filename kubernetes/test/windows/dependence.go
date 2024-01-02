package windows

import (
	k8sagent "boxroom/kubernetes/k8s-agent"
	k8sfilter "boxroom/kubernetes/k8s-filter"
	"boxroom/resource/immobile"
	"boxroom/resource/tree"
	utilfunc "boxroom/util/util-func"
	utillog "boxroom/util/util-log"
	mapset "github.com/deckarep/golang-set"
)

var log = new(utillog.NewLog).GetLogger()

func GetK8sWindowsAgent() (tree.Agent, error) {
	agent, err := (&k8sagent.ApiServerConfig{
		Url:              "https://192.168.100.140",
		Port:             "6443",
		KubernetesConfig: "/etc/kubernetes/admin.conf",
	}).AgentInit()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return agent, nil
}

func GetRootFilter(root *tree.KubernetesRoot) (*tree.KubernetesRoot, map[string]tree.Filter, error) {
	filters := map[string]tree.Filter{}
	rootFilter := &k8sfilter.KubernetesResourceFilter{
		Kind:              immobile.RootKind,
		ResourceInclude:   true,
		ResourceFilterSet: mapset.NewSet(),
	}
	filters[immobile.RootKind] = rootFilter
	if root != nil {
		newRoot := &tree.KubernetesRoot{}
		err := utilfunc.NewDuplicator().CopyFieldsByReflect(newRoot, *root)
		newRoot.Groups = map[string]*tree.Group{}
		if err != nil {
			return nil, nil, err
		}
		rootFilter.GetFilterSet().Add(newRoot)
	} else {
		rootFilter.GetFilterSet().Add(&tree.KubernetesRoot{
			Kind:     immobile.RootKind,
			Name:     immobile.RootName,
			TreeKind: immobile.TreeBackupKind,
			TreeName: "test-backup-crd-20230417212",
			Groups:   map[string]*tree.Group{},
		})
	}
	return &tree.KubernetesRoot{
		Kind:     immobile.RootKind,
		Name:     immobile.RootName,
		TreeKind: immobile.TreeBackupKind,
		TreeName: "test-backup-crd-20230417212",
		Groups:   map[string]*tree.Group{},
	}, filters, nil
}

func GetClusterResourceFilter() map[string]tree.Filter {
	filters := map[string]tree.Filter{}
	filter := &k8sfilter.KubernetesResourceFilter{
		Kind:              immobile.ClusterKind,
		ResourceInclude:   true,
		ResourceFilterSet: mapset.NewSet(),
	}
	filters[immobile.ClusterKind] = filter
	return filters
}
