package tree

import "github.io/misskaori/boxroom-crd/kubernetes/resource/immobile"

type KubernetesRoot struct {
	Kind     string
	Name     string
	TreeKind string
	TreeName string
	Parent   Resources
	Groups   map[string]*Group
}

func (root *KubernetesRoot) GetKind() string {
	return root.Kind
}

func (root *KubernetesRoot) GetName() string {
	return root.Name
}

func (root *KubernetesRoot) GetParent() Resources {
	return root.Parent
}

func (root *KubernetesRoot) GetChildren(name string) Resources {
	if !root.ContainsChildren(name) {
		return nil
	}
	return root.Groups[name]
}

func (root *KubernetesRoot) Equals(other Resources) bool {
	if root.Kind == other.GetKind() && root.Name == other.GetName() {
		return true
	}
	return false
}

func (root *KubernetesRoot) ContainsChildren(name string) bool {
	if _, ok := root.Groups[name]; !ok {
		return false
	}
	return true
}

func (root *KubernetesRoot) DeleteChildren(other Resources) bool {
	if !root.ContainsChildren(other.GetName()) {
		return false
	}
	delete(root.Groups, other.GetName())
	return true
}

func (root *KubernetesRoot) AddChildren(name string) *Group {
	if root.ContainsChildren(name) {
		return root.Groups[name]
	}
	child := &Group{
		Kind:     immobile.GroupKind,
		Name:     name,
		Parent:   root,
		Versions: map[string]*Version{},
	}
	root.Groups[name] = child
	return child
}

func (root *KubernetesRoot) ListChildren() []string {
	var childrenList []string
	for key := range root.Groups {
		childrenList = append(childrenList, key)
	}
	return childrenList
}
