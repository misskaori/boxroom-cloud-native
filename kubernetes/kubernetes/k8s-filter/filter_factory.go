package k8s_filter

import (
	mapset "github.com/deckarep/golang-set"
	"github.io/misskaori/boxroom-crd/kubernetes/resource/immobile"
	"github.io/misskaori/boxroom-crd/kubernetes/resource/tree"
)

func GetClusterResourceFilter(include bool) tree.Filter {
	filter := &KubernetesResourceFilter{
		Kind:              immobile.ClusterKind,
		ResourceInclude:   include,
		ResourceFilterSet: mapset.NewSet(),
	}

	return filter
}

func GetTreeRootFilter(clusterName, treeKind, treeName string) tree.Filter {
	filter := &KubernetesResourceFilter{
		Kind:              immobile.RootKind,
		ResourceInclude:   true,
		ResourceFilterSet: mapset.NewSet(),
	}

	root := &tree.KubernetesRoot{
		Kind:     immobile.RootKind,
		Name:     immobile.RootName,
		TreeKind: immobile.TreeBackupKind,
		Parent:   nil,
		Groups:   map[string]*tree.Group{},
	}

	filter.GetFilterSet().Add(root)

	return filter
}
