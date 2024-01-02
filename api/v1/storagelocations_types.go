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

package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StorageLocationsSpec defines the desired state of StorageLocations
type StorageLocationsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of StorageLocations. Edit storagelocations_types.go to remove/update
	ContainerSpec *StoragePluginContainerSpec `json:"containerSpec,omitempty"`
	ConfigSpec    *StoragePluginConfigSpec    `json:"configSpec,omitempty"`
}

type StoragePluginContainerSpec struct {
	Replicas int         `json:"replicas,omitempty"`
	Image    string      `json:"image,omitempty"`
	Port     int32       `json:"port,omitempty"`
	Protocol v1.Protocol `json:"protocol,omitempty"`
}

type StoragePluginConfigSpec struct {
	StorageKind   string                     `json:"storageKind,omitempty"`
	StorageConfig *StorageLocationSpecConfig `json:"config,omitempty"`
}

type StorageLocationSpecConfig struct {
	StorageUrl       string `json:"storageUrl,omitempty"`
	Region           string `json:"region,omitempty"`
	Bucket           string `json:"bucket,omitempty"`
	AccessKey        string `json:"accessKey,omitempty"`
	SecretKey        string `json:"secretKey,omitempty"`
	DisableSSL       bool   `json:"disableSSL,omitempty"`
	S3ForcePathStyle bool   `json:"s3ForcePathStyle,omitempty"`
}

// StorageLocationsStatus defines the observed state of StorageLocations
type StorageLocationsStatus struct {
	Replicas  int      `json:"replicas,omitempty"`
	Pods      []string `json:"pods,omitempty"`
	Service   string   `json:"service,omitempty"`
	ServiceIp string   `json:"serviceIp,omitempty"`
	ServicePort string `json:"servicePort,omitempty"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:statussd

// StorageLocations is the Schema for the storagelocations API
type StorageLocations struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageLocationsSpec   `json:"spec,omitempty"`
	Status StorageLocationsStatus `json:"status,omitempty"`
}

func (location *StorageLocations) GetPod() *v1.Pod {
	typeMeta := metav1.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	}

	objectMeta := metav1.ObjectMeta{
		GenerateName: location.Name + "-",
		Namespace:    location.Namespace,
		Labels:       map[string]string{"workload-kind": "storagelocations", "workload-name": location.Name},
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: location.APIVersion,
				Kind:       location.Kind,
				Name:       location.Name,
				UID:        location.UID,
			},
		},
	}

	spec := v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  "storage-plugin",
				Image: "liuweizhe/boxroom-s3-plugin:1.0-arm64",
				Ports: []v1.ContainerPort{
					{
						ContainerPort: location.Spec.ContainerSpec.Port,
						Protocol:      location.Spec.ContainerSpec.Protocol,
					},
				},

				Env: []v1.EnvVar{
					{
						Name:  "STORAGE_KIND",
						Value: location.Spec.ConfigSpec.StorageKind,
					},
					{
						Name:  "ACCESS_KEY",
						Value: location.Spec.ConfigSpec.StorageConfig.AccessKey,
					},
					{
						Name:  "SECRET_KEY",
						Value: location.Spec.ConfigSpec.StorageConfig.SecretKey,
					},
					{
						Name:  "ENDPOINT",
						Value: location.Spec.ConfigSpec.StorageConfig.StorageUrl,
					},
					{
						Name:  "REGION",
						Value: location.Spec.ConfigSpec.StorageConfig.Region,
					},
					{
						Name:  "BUCKET",
						Value: location.Spec.ConfigSpec.StorageConfig.Bucket,
					},
					{
						Name:  "DISABLE_SSL",
						Value: strconv.FormatBool(location.Spec.ConfigSpec.StorageConfig.DisableSSL),
					},
					{
						Name:  "S3_FORCE_PATH_STYLE",
						Value: strconv.FormatBool(location.Spec.ConfigSpec.StorageConfig.S3ForcePathStyle),
					},
				},
			},
		},
		RestartPolicy: v1.RestartPolicyAlways,
	}

	newPod := &v1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Spec:       spec,
	}

	return newPod
}

func (location *StorageLocations) GetService() *v1.Service {
	typeMeta := metav1.TypeMeta{
		Kind:       "Service",
		APIVersion: "v1",
	}

	objectMeta := metav1.ObjectMeta{
		Name:      location.Name + "-service",
		Namespace: location.Namespace,
		Labels:    map[string]string{"workload-kind": "storagelocations", "workload-name": location.Name},
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: location.APIVersion,
				Kind:       location.Kind,
				Name:       location.Name,
				UID:        location.UID,
			},
		},
	}

	spec := v1.ServiceSpec{
		Selector: map[string]string{"workload-kind": "storagelocations", "workload-name": location.Name},
		Ports: []v1.ServicePort{
			{
				Protocol: "TCP",
				Port:     8081,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8082,
				},
			},
		},
	}

	newService := &v1.Service{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Spec:       spec,
	}

	return newService
}

//+kubebuilder:object:root=true

// StorageLocationsList contains a list of StorageLocations
type StorageLocationsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageLocations `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StorageLocations{}, &StorageLocationsList{})
}
