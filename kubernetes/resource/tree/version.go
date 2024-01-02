package tree

import "github.io/misskaori/boxroom-crd/kubernetes/resource/immobile"

type Version struct {
	Kind      string
	Name      string
	Parent    *Group
	Resources map[string]*Resource
}

func (version *Version) GetKind() string {
	return version.Kind
}

func (version *Version) GetName() string {
	return version.Name
}

func (version *Version) GetParent() Resources {
	return version.Parent
}

func (version *Version) GetChildren(name string) Resources {
	if !version.ContainsChildren(name) {
		return nil
	}
	return version.Resources[name]
}

func (version *Version) Equals(other Resources) bool {
	if version.Kind == other.GetKind() && version.Name == other.GetName() {
		return true
	}
	return false
}

func (version *Version) ContainsChildren(name string) bool {
	if _, ok := version.Resources[name]; !ok {
		return false
	}
	return true
}

func (version *Version) DeleteChildren(other Resources) bool {
	if !version.ContainsChildren(other.GetName()) {
		return false
	}
	delete(version.Resources, other.GetName())
	return true
}

func (version *Version) AddChildren(name string, cluster bool) *Resource {
	if version.ContainsChildren(name) {
		return version.Resources[name]
	}
	child := &Resource{
		Kind:       immobile.ResourceKind,
		Name:       name,
		Parent:     version,
		IsCluster:  cluster,
		Namespaces: map[string]*Namespace{},
	}
	version.Resources[name] = child
	return child
}

func (version *Version) ListChildren() []string {
	var childrenList []string
	for key := range version.Resources {
		childrenList = append(childrenList, key)
	}
	return childrenList
}
