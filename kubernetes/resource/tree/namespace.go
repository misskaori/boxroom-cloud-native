package tree

import (
	"github.io/misskaori/boxroom-crd/kubernetes/resource/immobile"
)

type Namespace struct {
	Kind    string
	Name    string
	Parent  *Resource
	Objects map[string]*Object
}

func (namespace *Namespace) GetKind() string {
	return namespace.Kind
}

func (namespace *Namespace) GetName() string {
	return namespace.Name
}

func (namespace *Namespace) GetParent() Resources {
	return namespace.Parent
}

func (namespace *Namespace) GetChildren(name string) Resources {
	if !namespace.ContainsChildren(name) {
		return nil
	}
	return namespace.Objects[name]
}

func (namespace *Namespace) Equals(other Resources) bool {
	if namespace.Kind == other.GetKind() && namespace.Name == other.GetName() {
		return true
	}
	return false
}

func (namespace *Namespace) ContainsChildren(name string) bool {
	if _, ok := namespace.Objects[name]; !ok {
		return false
	}
	return true
}

func (namespace *Namespace) DeleteChildren(other Resources) bool {
	if !namespace.ContainsChildren(other.GetName()) {
		return false
	}
	delete(namespace.Objects, other.GetName())
	return true
}

func (namespace *Namespace) AddChildren(name string) *Object {
	if namespace.ContainsChildren(name) {
		return namespace.Objects[name]
	}
	child := &Object{
		Kind:   immobile.ObjectKind,
		Name:   name,
		Parent: namespace,
	}
	namespace.Objects[name] = child
	return child
}

func (namespace *Namespace) ListChildren() []string {
	var childrenList []string
	for key := range namespace.Objects {
		childrenList = append(childrenList, key)
	}
	return childrenList
}
