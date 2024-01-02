package tree

import (
	"context"
	mapset "github.com/deckarep/golang-set"
)

type AgentConfig interface {
	AgentInit() (Agent, error)
}

type Agent interface {
	GetResourceTree(root *KubernetesRoot, filters map[string]Filter, ctx context.Context) (*KubernetesRoot, error)
	ApplyResourceTree(root *KubernetesRoot, ctx context.Context) error
}

type Resources interface {
	GetKind() string
	GetName() string
	GetParent() Resources
	GetChildren(name string) Resources
	Equals(other Resources) bool
	ContainsChildren(name string) bool
	DeleteChildren(other Resources) bool
	ListChildren() []string
}

type Filter interface {
	GetFilterKind() string
	GetFilterPattern() bool
	GetFilterSet() mapset.Set
}

type Serialize interface {
	CovertStructToJson() ([]byte, error)
	CovertJsonToStruct(jsonDefinition []byte) error
}
