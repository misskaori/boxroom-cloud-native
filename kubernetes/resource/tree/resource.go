package tree

import "github.io/misskaori/boxroom-crd/kubernetes/resource/immobile"

type Resource struct {
	Kind       string
	Name       string
	Parent     *Version
	IsCluster  bool
	Namespaces map[string]*Namespace
}

func (resource *Resource) GetKind() string {
	return resource.Kind
}

func (resource *Resource) GetName() string {
	return resource.Name
}

func (resource *Resource) GetParent() Resources {
	return resource.Parent
}

func (resource *Resource) GetChildren(name string) Resources {
	if !resource.ContainsChildren(name) {
		return nil
	}
	return resource.Namespaces[name]
}

func (resource *Resource) Equals(other Resources) bool {
	if resource.Kind == other.GetKind() && resource.Name == other.GetName() {
		return true
	}
	return false
}

func (resource *Resource) ContainsChildren(name string) bool {
	if _, ok := resource.Namespaces[name]; !ok {
		return false
	}
	return true
}

func (resource *Resource) DeleteChildren(other Resources) bool {
	if !resource.ContainsChildren(other.GetName()) {
		return false
	}
	delete(resource.Namespaces, other.GetName())
	return true
}

func (resource *Resource) AddChildren(name string) *Namespace {
	if resource.ContainsChildren(name) {
		return resource.Namespaces[name]
	}
	child := &Namespace{
		Kind:    immobile.NamespaceKind,
		Name:    name,
		Parent:  resource,
		Objects: map[string]*Object{},
	}
	resource.Namespaces[name] = child
	return child
}

func (resource *Resource) ListChildren() []string {
	var childrenList []string
	for key := range resource.Namespaces {
		childrenList = append(childrenList, key)
	}
	return childrenList
}
