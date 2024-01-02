/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"github.com/golang/groupcache/lru"
	boxroomv1 "github.io/misskaori/boxroom-crd/api/v1"
	util_log "github.io/misskaori/boxroom-crd/kubernetes/util/util-log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sort"
	"strconv"
)

// StorageLocationsReconciler reconciles a StorageLocations object
var deletedPodCache = lru.New(1000)

type StorageLocationsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=boxroom.io,resources=storagelocations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=boxroom.io,resources=storagelocations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=boxroom.io,resources=storagelocations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the StorageLocations object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *StorageLocationsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	storageLocation, err := r.getStorageLocation(ctx, req)
	util_log.Logger.Infof("begin to handle storagelcoation: %v", storageLocation.Name)
	if storageLocation == nil {
		err = r.deleteBehaviour(ctx, req)
		if err != nil {
			util_log.Logger.Error(err)
		}
		return ctrl.Result{}, err
	}

	if err = r.syncBehaviour(ctx, storageLocation); err != nil {
		util_log.Logger.Error(err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *StorageLocationsReconciler) getStorageLocation(ctx context.Context, req ctrl.Request) (*boxroomv1.StorageLocations, error) {
	storageLocation := &boxroomv1.StorageLocations{}
	if err := r.Get(ctx, req.NamespacedName, storageLocation); err != nil {
		e := err.(errors.APIStatus)
		if e.Status().Reason == metav1.StatusReasonNotFound {
			return nil, nil
		}
	}
	if storageLocation.DeletionTimestamp != nil {
		return nil, nil
	}

	return storageLocation, nil
}

func (r *StorageLocationsReconciler) deleteBehaviour(ctx context.Context, req ctrl.Request) error {
	podList := &v1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(req.Namespace)); err != nil {
		return err
	}

	for _, pod := range podList.Items {
		if pod.OwnerReferences[0].Name == req.Name && pod.DeletionTimestamp == nil {
			util_log.Logger.Infof("begin to delete pod: %v", pod.Name)
			if err := r.deleteChildPod(ctx, &pod); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *StorageLocationsReconciler) syncBehaviour(ctx context.Context, storageLocation *boxroomv1.StorageLocations) error {
	util_log.Logger.Infof("begin to sync child pods: %v", storageLocation.Name)
	err := r.syncChildPodList(ctx, storageLocation)
	if err != nil {
		return err
	}
	util_log.Logger.Infof("begin to sync child services: %v", storageLocation.Name)
	err = r.syncChildService(ctx, storageLocation)
	if err != nil {
		return err
	}
	util_log.Logger.Infof("begin to update status: %v", storageLocation.Name)
	err = r.updateStorageLocation(ctx, storageLocation)
	return err
}

func (r *StorageLocationsReconciler) syncChildPodList(ctx context.Context, storageLocation *boxroomv1.StorageLocations) error {
	childPodList, err := r.getStorageLocationChildPodList(ctx, storageLocation)

	if err != nil {
		return err
	}

	n := storageLocation.Spec.ContainerSpec.Replicas - len(childPodList)
	createdOrDeletedChildPodNum := 0
	for createdOrDeletedChildPodNum != n {
		if n > 0 {
			if err := r.createChildPod(ctx, storageLocation); err == nil {
				createdOrDeletedChildPodNum++
			}
		}

		if n < 0 {
			if err := r.deleteChildPod(ctx, &childPodList[0]); err == nil {
				deletedPodCache.Add(childPodList[0].Name, "deleted")
				childPodList = childPodList[1:]
				createdOrDeletedChildPodNum--
			}
		}
	}

	childPodList, err = r.getStorageLocationChildPodList(ctx, storageLocation)
	if err != nil {
		return err
	}
	runningPodNum := 0
	var endPoints []string
	for _, pod := range childPodList {
		if pod.Status.Phase == v1.PodRunning {
			runningPodNum++
		}
		endPoints = append(endPoints, pod.Status.PodIP)
	}
	sort.Strings(endPoints)
	storageLocation.Status.Replicas = runningPodNum
	storageLocation.Status.Pods = endPoints

	return nil
}

func sortPodList(podList []v1.Pod) {
	l := len(podList)

	for i := 0; i < l-1; i++ {
		for j := 0; j < l-i-1; j++ {
			if podList[j].CreationTimestamp.Time.After(podList[j+1].CreationTimestamp.Time) {
				podList[j], podList[j+1] = podList[j+1], podList[j]
			}
		}
	}
}

func (r *StorageLocationsReconciler) syncChildService(ctx context.Context, storageLocation *boxroomv1.StorageLocations) error {
	childServiceList, err := r.getStorageLocationChildServiceList(ctx, storageLocation)
	if err != nil {
		return err
	}

	n := 1 - len(childServiceList)
	createdOrDeletedChildServiceNum := 0
	for createdOrDeletedChildServiceNum != n {
		if n > 0 {
			if err := r.createChildService(ctx, storageLocation); err == nil {
				createdOrDeletedChildServiceNum++
			}
		}

		if n < 0 {
			if err := r.deleteChildService(ctx, &childServiceList[0]); err == nil {
				childServiceList = childServiceList[1:]
				createdOrDeletedChildServiceNum--
			}
		}
	}

	childServiceList, err = r.getStorageLocationChildServiceList(ctx, storageLocation)
	if err != nil {
		return err
	}

	if len(childServiceList) != 1 {
		err = r.syncChildService(ctx, storageLocation)
		if err != nil {
			return err
		}
	} else {
		storageLocation.Status.Service = childServiceList[0].Name
		storageLocation.Status.ServiceIp = childServiceList[0].Spec.ClusterIP
		storageLocation.Status.ServicePort = strconv.Itoa(int(childServiceList[0].Spec.Ports[0].Port))
	}

	return nil
}

func (r *StorageLocationsReconciler) createChildService(ctx context.Context, storageLocation *boxroomv1.StorageLocations) error {
	if err := r.Create(ctx, storageLocation.GetService()); err != nil {
		return err
	}
	return nil
}

func (r *StorageLocationsReconciler) deleteChildService(ctx context.Context, service *v1.Service) error {
	if err := r.Delete(ctx, service); err != nil {
		return err
	}
	return nil
}

func (r *StorageLocationsReconciler) createChildPod(ctx context.Context, storageLocation *boxroomv1.StorageLocations) error {
	if err := r.Create(ctx, storageLocation.GetPod()); err != nil {
		return err
	}
	return nil
}

func (r *StorageLocationsReconciler) deleteChildPod(ctx context.Context, pod *v1.Pod) error {
	if err := r.Delete(ctx, pod); err != nil {
		return err
	}
	return nil
}

func (r *StorageLocationsReconciler) updateStorageLocation(ctx context.Context, storageLocation *boxroomv1.StorageLocations) error {
	newStorageLocation, err := r.getStorageLocation(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: storageLocation.Namespace,
			Name:      storageLocation.Name,
		},
	})
	if err != nil {
		return err
	}

	if reflect.DeepEqual(newStorageLocation.Status, storageLocation.Status) {
		util_log.Logger.Infof("storagelocation %v update status: storagelocation status is same as old", storageLocation.Name)
		return nil
	}

	if newStorageLocation.ObjectMeta.Generation > storageLocation.ObjectMeta.Generation {
		util_log.Logger.Infof("storagelocation %v update status: new version's generation is greater than old version's", storageLocation.Name)
		return nil
	}

	if err = r.Update(ctx, storageLocation); err != nil {
		return err
	}
	util_log.Logger.Infof("storagelocation %v update status: successful update status, old is %v, new is %v", storageLocation.Name, newStorageLocation.Status, storageLocation.Status)
	return nil
}

func (r *StorageLocationsReconciler) getStorageLocationChildPodList(ctx context.Context, storageLocation *boxroomv1.StorageLocations) ([]v1.Pod, error) {
	podList := &v1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(storageLocation.Namespace)); err != nil {
		util_log.Logger.Error(err)
		return nil, err
	}

	var childPodList []v1.Pod
	for _, pod := range podList.Items {
		if _, alive := deletedPodCache.Get(pod.Name); !alive && pod.Labels["workload-kind"] == "storagelocations" && pod.OwnerReferences[0].Name == storageLocation.Name && pod.OwnerReferences[0].UID == storageLocation.UID && pod.DeletionTimestamp == nil {
			childPodList = append(childPodList, pod)
		}
	}

	sortPodList(childPodList)
	return childPodList, nil
}

func (r *StorageLocationsReconciler) getStorageLocationChildServiceList(ctx context.Context, storageLocation *boxroomv1.StorageLocations) ([]v1.Service, error) {
	serviceList := &v1.ServiceList{}
	if err := r.List(ctx, serviceList, client.InNamespace(storageLocation.Namespace)); err != nil {
		util_log.Logger.Error(err)
		return nil, err
	}

	var childServiceList []v1.Service
	for _, service := range serviceList.Items {
		if service.Labels["workload-kind"] == "storagelocations" && service.OwnerReferences[0].Name == storageLocation.Name && service.OwnerReferences[0].UID == storageLocation.UID && service.DeletionTimestamp == nil {
			childServiceList = append(childServiceList, service)
		}
	}

	return childServiceList, nil
}

type EnqueueRequestForStorageLocationChildren struct {
}

func (e *EnqueueRequestForStorageLocationChildren) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Object.GetLabels()["workload-kind"] != "storagelocations" {
		return
	}
	q.Add(reconcile.Request{NamespacedName: *e.getStorageLocationRequestFromObject(evt.Object)})
}

// Update implements EventHandler.
func (e *EnqueueRequestForStorageLocationChildren) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.ObjectNew.GetLabels()["workload-kind"] != "storagelocations" {
		return
	}
	//q.Add(reconcile.Request{NamespacedName: *e.getStorageLocationRequestFromObject(evt.ObjectOld)})
	q.Add(reconcile.Request{NamespacedName: *e.getStorageLocationRequestFromObject(evt.ObjectNew)})
}

// Delete implements EventHandler.
func (e *EnqueueRequestForStorageLocationChildren) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Object.GetLabels()["workload-kind"] != "storagelocations" {
		return
	}
	q.Add(reconcile.Request{NamespacedName: *e.getStorageLocationRequestFromObject(evt.Object)})
}

// Generic implements EventHandler.
func (e *EnqueueRequestForStorageLocationChildren) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if evt.Object.GetLabels()["workload-kind"] != "storagelocations" {
		return
	}
	q.Add(reconcile.Request{NamespacedName: *e.getStorageLocationRequestFromObject(evt.Object)})
}

func (e *EnqueueRequestForStorageLocationChildren) getStorageLocationRequestFromObject(object client.Object) *types.NamespacedName {
	if len(object.GetOwnerReferences()) == 0 {
		return nil
	}
	storagelocation := object.GetOwnerReferences()[0]
	return &types.NamespacedName{
		Name:      storagelocation.Name,
		Namespace: object.GetNamespace(),
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *StorageLocationsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&boxroomv1.StorageLocations{}).Watches(&v1.Pod{}, &EnqueueRequestForStorageLocationChildren{}).Watches(&v1.Service{}, &EnqueueRequestForStorageLocationChildren{}).Complete(r)
}
