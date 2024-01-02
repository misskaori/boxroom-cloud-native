package tree

import (
	"encoding/json"
	util_func "github.io/misskaori/boxroom-crd/kubernetes/util/util-func"
	utillog "github.io/misskaori/boxroom-crd/kubernetes/util/util-log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var log = new(utillog.NewLog).GetLogger()

type Object struct {
	Kind       string
	Name       string
	Parent     *Namespace
	GVR        *schema.GroupVersionResource
	Definition *unstructured.Unstructured
	Metadata   *ObjectMetadata
}

func (object *Object) GetKind() string {
	return object.Kind
}

func (object *Object) GetName() string {
	return object.Name
}

func (object *Object) GetParent() Resources {
	return object.Parent
}

func (object *Object) GetChildren(name string) Resources {
	return nil
}

func (object *Object) Equals(other Resources) bool {
	if object.Kind == other.GetKind() && object.Name == other.GetName() {
		return true
	}
	return false
}

func (object *Object) ContainsChildren(kind string) bool {
	if _, ok := object.Definition.Object[kind]; !ok {
		return false
	}
	return true
}

func (object *Object) DeleteChildren(other Resources) bool {
	if !object.ContainsChildren(other.GetKind()) {
		return false
	}
	delete(object.Definition.Object, other.GetKind())
	return true
}

func (object *Object) CovertStructToJson() ([]byte, error) {
	jsonDefinition, err := object.Definition.MarshalJSON()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return jsonDefinition, nil
}

func (object *Object) CovertJsonToStruct(jsonDefinition []byte) error {
	definition := &unstructured.Unstructured{}
	err := definition.UnmarshalJSON(jsonDefinition)
	if err != nil {
		log.Error(err)
		return err
	}
	object.Definition = definition
	return nil
}

func (object *Object) ListChildren() []string {
	var childrenList []string
	return childrenList
}

type ObjectMetadata struct {
	Kind      string
	Name      string
	Group     string
	Version   string
	Resource  string
	Namespace string
	IsCluster bool
}

func (metadata *ObjectMetadata) CovertStructToJson() ([]byte, error) {
	jsonDefinition, err := json.Marshal(metadata)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return jsonDefinition, nil
}

func (metadata *ObjectMetadata) CovertJsonToStruct(jsonDefinition []byte) error {
	var objectMetadata = &ObjectMetadata{}
	err := json.Unmarshal(jsonDefinition, objectMetadata)
	if err != nil {
		log.Error(err)
		return err
	}
	err = util_func.NewDuplicator().CopyFieldsByReflect(metadata, *objectMetadata)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
