package k8s_agent

import (
	"context"
	"errors"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"github.com/sirupsen/logrus"
	k8sfilter "github.io/misskaori/boxroom-crd/kubernetes/kubernetes/k8s-filter"
	"github.io/misskaori/boxroom-crd/kubernetes/resource/immobile"
	"github.io/misskaori/boxroom-crd/kubernetes/resource/tree"
	"github.io/misskaori/boxroom-crd/kubernetes/util/globle-immobile"
	utilfunc "github.io/misskaori/boxroom-crd/kubernetes/util/util-func"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
	anotherv1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"strings"
)

type KubernetesAgent struct {
	ConfigObject    *restclient.Config
	DynamicClient   *dynamic.DynamicClient
	DiscoveryClient *discovery.DiscoveryClient
	ClientSet       *kubernetes.Clientset
}

func (client *KubernetesAgent) GetResourceTree(root *tree.KubernetesRoot, filters map[string]tree.Filter, ctx context.Context) (*tree.KubernetesRoot, error) {
	fileLoggerValue := ctx.Value(globle_immobile.FileLogger)
	if fileLoggerValue == nil {
		return nil, errors.New("there is no file logger which is request when get resource tree")
	}
	fileLogger := fileLoggerValue.(*logrus.Logger)

	if filters == nil {
		filters = map[string]tree.Filter{}
	}

	fileLogger.Infof("begin to init tree")
	_, vs, err := client.DiscoveryClient.ServerGroupsAndResources()
	if err != nil {
		fileLogger.Error(err)
		return nil, err
	}
	ns, err := client.ClientSet.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fileLogger.Error(err)
		return nil, err
	}

	fileLogger.Infof("filt tree: begin to filt tree, there are %d tree filters", len(filters))
	preHandleNamespaceFilter(filters)
	preHandleResourceFilter(filters)
	clusterInclude := false

	for key, f := range filters {
		fileLogger.Infof("filt resouce: The kind of this s3-filter is %s ", f.GetFilterKind())
		switch key {
		case immobile.ResourceKind:
			filtrateResource(vs, f, ctx)
			delete(filters, immobile.ResourceKind)
		case immobile.NamespaceKind:
			filtrateNamespace(ns, f, ctx)
			delete(filters, immobile.NamespaceKind)
		case immobile.ClusterKind:
			if f.GetFilterKind() == immobile.ClusterKind && f.GetFilterPattern() {
				clusterInclude = true
			}
		}
	}
	fileLogger.Infof("begin to build resouece tree")
	return client.buildGroupAndVersionTree(root, vs, ns, clusterInclude, ctx)
}

func (client *KubernetesAgent) ApplyResourceTree(root *tree.KubernetesRoot, ctx context.Context) error {
	fileLoggerValue := ctx.Value(globle_immobile.FileLogger)
	if fileLoggerValue == nil {
		return errors.New("there is no file logger which is request when get resource tree")
	}
	fileLogger := fileLoggerValue.(*logrus.Logger)

	filters := map[string]tree.Filter{}

	filters[immobile.ClusterKind] = k8sfilter.GetClusterResourceFilter(true)

	fileLogger.Info("start to get integrated resource tree")
	currentTreeRoot, err := client.GetResourceTree(&tree.KubernetesRoot{
		Kind:     immobile.RootKind,
		Name:     immobile.RootName,
		TreeKind: immobile.TreeBackupKind,
		TreeName: "compare",
		Parent:   nil,
		Groups:   map[string]*tree.Group{},
	}, filters, ctx)
	if err != nil {
		fileLogger.Error(err)
		return err
	}

	fileLogger.Info("integrated resource tree is completed")

	fileLogger.Info("start to compare integrated resource tree with backup resource tree and restore objects")
	err = client.compareGroupTree(root, currentTreeRoot, ctx)
	if err != nil {
		fileLogger.Error(err)
		return err
	}
	if len(root.Groups) == 0 {
		fileLogger.Info("there is no resource that need to be restored")
	}

	return nil
}

func (client *KubernetesAgent) compareGroupTree(backupTreeRoot, currentTreeRoot *tree.KubernetesRoot, ctx context.Context) error {
	fileLogger, _ := ctx.Value(globle_immobile.FileLogger).(*logrus.Logger)

	for backupGroupName, backupGroup := range backupTreeRoot.Groups {
		currentGroup := currentTreeRoot.AddChildren(backupGroupName)
		err := client.compareVersionTree(backupGroup, currentGroup, ctx)
		if err != nil {
			fileLogger.Error(err)
			continue
		}
		if len(backupGroup.Versions) == 0 {
			backupTreeRoot.DeleteChildren(backupGroup)
		}
	}
	return nil
}

func (client *KubernetesAgent) compareVersionTree(backupGroup, currentGroup *tree.Group, ctx context.Context) error {
	fileLogger, _ := ctx.Value(globle_immobile.FileLogger).(*logrus.Logger)

	for backupVersionName, backupVersion := range backupGroup.Versions {
		currentVersion := currentGroup.AddChildren(backupVersionName)
		err := client.compareResourceTree(backupVersion, currentVersion, ctx)
		if err != nil {
			fileLogger.Error(err)
			continue
		}
		if len(backupVersion.Resources) == 0 {
			backupGroup.DeleteChildren(backupVersion)
		}
	}
	return nil
}

func (client *KubernetesAgent) compareResourceTree(backupVersion, currentVersion *tree.Version, ctx context.Context) error {
	fileLogger, _ := ctx.Value(globle_immobile.FileLogger).(*logrus.Logger)

	for backupResourceName, backupResource := range backupVersion.Resources {
		currentResource := currentVersion.AddChildren(backupResourceName, backupResource.IsCluster)
		err := client.restoreNamespaceTree(backupResource, currentResource, ctx)
		if err != nil {
			fileLogger.Error(err)
			continue
		}
		if len(backupResource.Namespaces) == 0 {
			backupVersion.DeleteChildren(backupResource)
		}
	}
	return nil
}

func (client *KubernetesAgent) restoreNamespaceTree(backupResource, currentResource *tree.Resource, ctx context.Context) error {
	fileLogger, _ := ctx.Value(globle_immobile.FileLogger).(*logrus.Logger)

	for backupNamespaceName, backupNamespace := range backupResource.Namespaces {
		if _, ok := currentResource.Namespaces[backupNamespaceName]; !ok && backupNamespaceName != immobile.ClusterLevelNamespace {
			err := client.applyNamespace(backupNamespace, ctx)
			if err != nil {
				fileLogger.Error(err)
				continue
			}
		}
		currentNamespace := currentResource.AddChildren(backupNamespaceName)
		err := client.restoreObjectTree(backupNamespace, currentNamespace, ctx)
		if err != nil {
			fileLogger.Error(err)
			continue
		}
		if len(backupNamespace.Objects) == 0 {
			backupResource.DeleteChildren(backupNamespace)
		}
	}
	return nil
}

func (client *KubernetesAgent) restoreObjectTree(backupNamespace, currentNamespace *tree.Namespace, ctx context.Context) error {
	fileLogger, _ := ctx.Value(globle_immobile.FileLogger).(*logrus.Logger)

	for backupObjectName, backupObject := range backupNamespace.Objects {
		if _, ok := currentNamespace.Objects[backupObjectName]; !ok {
			err := client.applyObject(backupObject, ctx)
			if err != nil {
				fileLogger.Error(err)
				continue
			}
		} else {
			backupNamespace.DeleteChildren(backupObject)
		}
	}

	return nil
}

func (client *KubernetesAgent) applyNamespace(namespace *tree.Namespace, ctx context.Context) error {
	fileLogger, _ := ctx.Value(globle_immobile.FileLogger).(*logrus.Logger)

	namespaceResourceKind := "Namespace"
	namespaceResourceVersion := "v1"
	namespaceInput := &corev1.NamespaceApplyConfiguration{
		TypeMetaApplyConfiguration: anotherv1.TypeMetaApplyConfiguration{
			Kind:       &namespaceResourceKind,
			APIVersion: &namespaceResourceVersion,
		},

		ObjectMetaApplyConfiguration: &anotherv1.ObjectMetaApplyConfiguration{
			Name: &namespace.Name,
		},
	}

	_, err := client.ClientSet.CoreV1().Namespaces().Apply(context.TODO(), namespaceInput, metav1.ApplyOptions{FieldManager: "application/apply-patch"})

	if err != nil {
		fileLogger.Error(err)
		return err
	}
	return nil
}

func (client *KubernetesAgent) applyObject(object *tree.Object, ctx context.Context) error {
	fileLogger, _ := ctx.Value(globle_immobile.FileLogger).(*logrus.Logger)
	missionStatus, _ := ctx.Value(globle_immobile.MissionStatus).(tree.Status)

	objectPathName := utilfunc.NewWorkDirOperator().GenerateDirPath(object.Metadata.Group, object.Metadata.Version, object.Metadata.Resource, object.Metadata.Namespace, object.Metadata.Name)

	fileLogger.Infof("restore objects: kind: %s name: %s", object.Metadata.Resource, object.Metadata.Name)
	if object.Definition == nil {
		e := fmt.Sprintf("there is no definition of this object: namespace:%s kind:%s name:%s", object.Parent.Name, object.Kind, object.Name)
		fileLogger.Error(e)
		return errors.New(e)
	}

	preHandleObjectBeforeCreate(object)

	if object.Metadata.IsCluster {
		_, err := client.DynamicClient.Resource(*object.GVR).Apply(context.TODO(), object.GetName(), object.Definition, metav1.ApplyOptions{
			FieldManager: "cluster-manager",
		})
		if err != nil {
			fileLogger.Error(err)
			missionStatus.SetStatus(StatusPartialFailed)
			missionStatus.AddFailedObjects(objectPathName, err)
			return err
		}
	} else {
		_, err := client.DynamicClient.Resource(*object.GVR).Namespace(object.Metadata.Namespace).Create(context.TODO(), object.Definition, metav1.CreateOptions{})
		statusErr, _ := err.(k8serrors.APIStatus)
		if err != nil && statusErr.Status().Reason != metav1.StatusReasonAlreadyExists {
			fileLogger.Error(err)
			missionStatus.SetStatus(StatusPartialFailed)
			missionStatus.AddFailedObjects(objectPathName, err)
			return err
		} else if statusErr != nil {
			fileLogger.Info(err)
		} else {
			fileLogger.Infof("restore success: namespace: %s resource: %s object: %s", object.Metadata.Namespace, object.Metadata.Resource, object.Metadata.Name)
		}
	}

	return nil
}

func (client *KubernetesAgent) buildGroupAndVersionTree(root *tree.KubernetesRoot, groupAndVersions []*metav1.APIResourceList, namespaces *v1.NamespaceList, clusterInclude bool, ctx context.Context) (*tree.KubernetesRoot, error) {
	fileLogger, _ := ctx.Value(globle_immobile.FileLogger).(*logrus.Logger)

	for _, groupAndVersion := range groupAndVersions {
		gv, err := schema.ParseGroupVersion(groupAndVersion.GroupVersion)
		if err != nil {
			fileLogger.Error(err)
			return nil, err
		}
		v := root.AddChildren(gv.Group).AddChildren(gv.Version)
		err = client.buildResourceTree(v, &gv, groupAndVersion, namespaces, clusterInclude, ctx)
		if err != nil {
			return nil, err
		}
		if len(v.Resources) == 0 && !v.Parent.DeleteChildren(v) {
			e := fmt.Sprintf("there is no such version to delete: %v", v)
			return nil, errors.New(e)
		}
	}
	for _, group := range root.Groups {
		if len(group.Versions) == 0 && !root.DeleteChildren(group) {
			e := fmt.Sprintf("there is no such group to delete: %v", group)
			return nil, errors.New(e)
		}
	}
	return root, nil
}

func (client *KubernetesAgent) buildResourceTree(version *tree.Version, gv *schema.GroupVersion, groupAndVersion *metav1.APIResourceList, namespaces *v1.NamespaceList, clusterInclude bool, ctx context.Context) error {
	fileLogger, _ := ctx.Value(globle_immobile.FileLogger).(*logrus.Logger)

	for _, api := range groupAndVersion.APIResources {
		if !clusterInclude && !api.Namespaced && !defaultIncludeClusterResource().Contains(api.Name) {
			continue
		}
		r := version.AddChildren(api.Name, !api.Namespaced)
		gvr := schema.GroupVersionResource{
			Group:    gv.Group,
			Version:  gv.Version,
			Resource: api.Name,
		}
		fileLogger.Infof("build resource branches: group:%s version:%s resource:%s", gvr.Group, gvr.Version, gvr.Resource)
		err := client.buildNamespaceTree(r, &gvr, namespaces, ctx)
		if err != nil {
			fileLogger.Error(err)
			return err
		}
	}
	for _, resource := range version.Resources {
		if len(resource.Namespaces) == 0 && !version.DeleteChildren(resource) {
			e := fmt.Sprintf("there is no such resource to delete: %v", resource)
			return errors.New(e)
		}
	}
	return nil
}

func (client *KubernetesAgent) buildNamespaceTree(resource *tree.Resource, gvr *schema.GroupVersionResource, namespaces *v1.NamespaceList, ctx context.Context) error {
	if resource.IsCluster {
		resource.AddChildren(immobile.ClusterLevelNamespace)
	} else {
		for _, namespace := range namespaces.Items {
			resource.AddChildren(namespace.Name)
		}
	}
	_ = client.buildObjectTree(resource, gvr, ctx)
	for _, namespace := range resource.Namespaces {
		if len(namespace.Objects) == 0 && !resource.DeleteChildren(namespace) {
			e := fmt.Sprintf("there is no such namespace to delete: %v", namespace)
			return errors.New(e)
		}
	}
	return nil
}

func (client *KubernetesAgent) buildObjectTree(resource *tree.Resource, gvr *schema.GroupVersionResource, ctx context.Context) error {
	fileLogger, _ := ctx.Value(globle_immobile.FileLogger).(*logrus.Logger)

	unstructObj, err := client.DynamicClient.Resource(*gvr).List(context.TODO(), metav1.ListOptions{})
	filtFlag := false
	objectFilter := defaultObjectFilter()
	if _, ok := objectFilter[gvr.Resource]; ok {
		filtFlag = true
	}

	if err != nil {
		//TODO:HANDLE THIS ERROR MORE GRACEFULLY
		fileLogger.Infof("could not found the requested object: group:%s, version:%s, resource:%s native error info:%s", gvr.Group, gvr.Version, gvr.Resource, err.Error())
		return err
	}
	for idx, object := range unstructObj.Items {

		if filtFlag && !preHandleObjectFilter(&object, gvr) {
			continue
		}

		namespaceName := object.GetNamespace()

		if resource.IsCluster {
			namespaceName = immobile.ClusterLevelNamespace
		}
		if resource.ContainsChildren(namespaceName) {
			fileLogger.Infof("build resource leaves: kind:%s namespace:%s object:%s", gvr.Resource, namespaceName, object.GetName())
			namespace := resource.Namespaces[namespaceName]
			namespace.AddChildren(object.GetName())
			treeObject := namespace.Objects[object.GetName()]
			treeObject.GVR = gvr
			treeObject.Definition = &unstructObj.Items[idx]
			treeObject.Metadata = &tree.ObjectMetadata{
				Kind:      treeObject.Kind,
				Name:      treeObject.Name,
				Group:     treeObject.GVR.Group,
				Version:   treeObject.GVR.Version,
				Resource:  treeObject.GVR.Resource,
				IsCluster: treeObject.Parent.Parent.IsCluster,
				Namespace: namespace.Name,
			}
		}
	}
	return nil
}

func filtrateNamespace(namespaces *v1.NamespaceList, filter tree.Filter, ctx context.Context) {
	fileLogger, _ := ctx.Value(globle_immobile.FileLogger).(*logrus.Logger)

	for i := 0; i < len(namespaces.Items); {
		addflag := (filter.GetFilterPattern() && filter.GetFilterSet().Contains(namespaces.Items[i].Name)) || (!filter.GetFilterPattern() && !filter.GetFilterSet().Contains(namespaces.Items[i].Name))
		if !addflag {
			fileLogger.Infof("filt resouce: kind:namespace name:%s handle:excluded", namespaces.Items[i].Name)
			namespaces.Items = append(namespaces.Items[:i], namespaces.Items[i+1:]...)
		} else {
			fileLogger.Infof("filt resouce: kind:namespace name:%s handle:incuded", namespaces.Items[i].Name)
			i++
		}
	}
}

func filtrateResource(groupAndVersions []*metav1.APIResourceList, filter tree.Filter, ctx context.Context) {
	fileLogger, _ := ctx.Value(globle_immobile.FileLogger).(*logrus.Logger)

	for j := 0; j < len(groupAndVersions); {
		version := groupAndVersions[j]
		for i := 0; i < len(version.APIResources); {
			addflag := (filter.GetFilterPattern() && filter.GetFilterSet().Contains(version.APIResources[i].Name)) || (!filter.GetFilterPattern() && !filter.GetFilterSet().Contains(version.APIResources[i].Name))
			if !addflag {
				fileLogger.Infof("filt resouce: kind:tree name:%s handle:excluded", version.APIResources[i].Name)
				version.APIResources = append(version.APIResources[:i], version.APIResources[i+1:]...)
			} else {
				fileLogger.Infof("filt resouce: kind:tree name:%s handle:included", version.APIResources[i].Name)
				i++
			}
		}
		if len(version.APIResources) == 0 {
			groupAndVersions = append(groupAndVersions[:j], groupAndVersions[j+1:]...)
		} else {
			j++
		}
	}
}

func preHandleNamespaceFilter(filters map[string]tree.Filter) {
	defaultFilter := defaultNamespaceFilter()
	if filter, ok := filters[immobile.NamespaceKind]; ok {
		if !filter.GetFilterPattern() {
			for ns := range defaultFilter.GetFilterSet().Iter() {
				filter.GetFilterSet().Add(ns)
			}
		} else {
			for ns := range defaultFilter.GetFilterSet().Iter() {
				filter.GetFilterSet().Remove(ns)
			}
		}
	} else if !ok {
		filters[immobile.NamespaceKind] = defaultFilter
	}
}

func preHandleResourceFilter(filters map[string]tree.Filter) {
	defaultFilter := defaultResourceFilter()
	if filter, ok := filters[immobile.ResourceKind]; ok {
		if !filter.GetFilterPattern() {
			for ns := range defaultFilter.GetFilterSet().Iter() {
				filter.GetFilterSet().Add(ns)
			}
		} else {
			for ns := range defaultFilter.GetFilterSet().Iter() {
				filter.GetFilterSet().Remove(ns)
			}
		}
	} else if !ok {
		filters[immobile.ResourceKind] = defaultResourceFilter()
	}
}

func preHandleObjectFilter(object *unstructured.Unstructured, gvr *schema.GroupVersionResource) bool {
	addFlag := true
	defaultObjectFilter := defaultObjectFilter()
	if filter, ok := defaultObjectFilter[gvr.Resource]; ok {
		for element := range filter.GetFilterSet().Iter() {
			substring := element.(string)
			if strings.Contains(object.GetName(), substring) {
				addFlag = false
			}
		}
	}
	return addFlag
}

func defaultNamespaceFilter() tree.Filter {
	filter := &k8sfilter.KubernetesResourceFilter{
		Kind:              immobile.NamespaceKind,
		ResourceInclude:   false,
		ResourceFilterSet: mapset.NewSet(),
	}

	list := []string{"kube-system", "kube-public", "kube-node-lease"}

	for _, ns := range list {
		filter.ResourceFilterSet.Add(ns)
	}
	return filter
}

func defaultResourceFilter() tree.Filter {
	filter := &k8sfilter.KubernetesResourceFilter{
		Kind:              immobile.ResourceKind,
		ResourceInclude:   false,
		ResourceFilterSet: mapset.NewSet(),
	}

	list := []string{"events", "pods", "endpointslices"}

	for _, ns := range list {
		filter.ResourceFilterSet.Add(ns)
	}
	return filter
}

func defaultObjectFilter() map[string]tree.Filter {
	secret := "secrets"

	objectFilters := map[string]tree.Filter{}

	secretFilter := &k8sfilter.KubernetesResourceFilter{
		Kind:              secret,
		ResourceInclude:   false,
		ResourceFilterSet: mapset.NewSet(),
	}

	secretList := []string{"default-token"}

	for _, object := range secretList {
		secretFilter.ResourceFilterSet.Add(object)
	}

	objectFilters[secret] = secretFilter

	return objectFilters
}

func defaultIncludeClusterResource() mapset.Set {
	defaultIncludeResource := []string{"namespaces", "persistentvolumes"}

	set := mapset.NewSet()
	for _, resource := range defaultIncludeResource {
		set.Add(resource)
	}

	return set
}

func preHandleObjectBeforeCreate(object *tree.Object) {
	delete(object.Definition.Object, "status")

	metadataFiled, _ := object.Definition.Object["metadata"].(map[string]interface{})
	delete(metadataFiled, "managedFields")
	delete(metadataFiled, "uid")
	delete(metadataFiled, "resourceVersion")
	delete(metadataFiled, "creationTimestamp")
	delete(metadataFiled, "selfLink")
	delete(metadataFiled, "generation")
	delete(metadataFiled, "resourceVersion")
	delete(metadataFiled, "ownerReferences")

	specFiled, _ := object.Definition.Object["spec"].(map[string]interface{})
	delete(specFiled, "claimRef")
}
