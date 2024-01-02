package tree

import "github.io/misskaori/boxroom-crd/kubernetes/resource/immobile"

type Group struct {
	Kind     string
	Name     string
	Parent   *KubernetesRoot
	Versions map[string]*Version
}

func (group *Group) GetKind() string {
	return group.Kind
}

func (group *Group) GetName() string {
	return group.Name
}

func (group *Group) GetParent() Resources {
	return group.Parent
}

func (group *Group) GetChildren(name string) Resources {
	if !group.ContainsChildren(name) {
		return nil
	}
	return group.Versions[name]
}

func (group *Group) Equals(other Resources) bool {
	if group.Kind == other.GetKind() && group.Name == other.GetName() {
		return true
	}
	return false
}

func (group *Group) ContainsChildren(name string) bool {
	if _, ok := group.Versions[name]; !ok {
		return false
	}
	return true
}

func (group *Group) DeleteChildren(other Resources) bool {
	if !group.ContainsChildren(other.GetName()) {
		return false
	}
	delete(group.Versions, other.GetName())
	return true
}

func (group *Group) AddChildren(name string) *Version {
	if group.ContainsChildren(name) {
		return group.Versions[name]
	}
	child := &Version{
		Kind:      immobile.VersionKind,
		Name:      name,
		Parent:    group,
		Resources: map[string]*Resource{},
	}
	group.Versions[name] = child
	return child
}

func (group *Group) ListChildren() []string {
	var childrenList []string
	for key := range group.Versions {
		childrenList = append(childrenList, key)
	}
	return childrenList
}
