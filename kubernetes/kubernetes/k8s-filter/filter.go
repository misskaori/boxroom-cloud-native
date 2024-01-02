package k8s_filter

import (
	mapset "github.com/deckarep/golang-set"
)

type KubernetesResourceFilter struct {
	Kind              string
	ResourceInclude   bool
	ResourceFilterSet mapset.Set
}

func (filter *KubernetesResourceFilter) GetFilterKind() string {
	return filter.Kind
}

func (filter *KubernetesResourceFilter) GetFilterPattern() bool {
	return filter.ResourceInclude
}

func (filter *KubernetesResourceFilter) GetFilterSet() mapset.Set {
	return filter.ResourceFilterSet
}
