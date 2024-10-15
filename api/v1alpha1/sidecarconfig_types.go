/*
Copyright 2024.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// InjectConfig defines the configuration for the sidecar injection
type InjectConfig struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// selector is a label query over pods that should be injected
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// InjectKidecar indicates whether to inject the kidecar instead of user define
	InjectKidecar bool `json:"injectSidecar,omitempty"`

	// UseKubeNativeSidecar indicates whether to use the kube native sidecar, kube version must higher than 1.28
	UseKubeNativeSidecar bool `json:"useKubeNativeSidecar,omitempty"`

	// Namespace sidecarSet will only match the pods in the namespace
	// otherwise, match pods in all namespaces(in cluster)
	Namespace string `json:"namespace,omitempty"`

	// NamespaceSelector select which namespaces to inject sidecar containers.
	// Default to the empty LabelSelector, which matches everything.
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// InitContainers is the list of init containers to be injected into the selected pod
	// We will inject those containers by their name in ascending order
	// We only inject init containers when a new pod is created, it does not apply to any existing pod
	// +patchMergeKey=name
	// +patchStrategy=merge
	InitContainers []corev1.Container `json:"initContainers,omitempty" patchStrategy:"merge" patchMergeKey:"name"`

	// Containers is the list of sidecar containers to be injected into the selected pod
	// +patchMergeKey=name
	// +patchStrategy=merge
	Containers []corev1.Container `json:"containers,omitempty" patchStrategy:"merge" patchMergeKey:"name"`

	// List of volumes that can be mounted by sidecar containers
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +patchMergeKey=name
	// +patchStrategy=merge
	Volumes []corev1.Volume `json:"volumes,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	// VolumeMounts is the list of VolumeMounts that can be mounted by sidecar containers
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +patchMergeKey=name
	// +patchStrategy=merge
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
	// ForceInjectServiceAccount indicates whether to inject the service account to the pod forcely
	ForceInjectServiceAccount *bool `json:"forceInjectServiceAccount,omitempty"`
	// ServiceAccountName is the name of the service account to inject to the pod
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// ShareProcessNamespace indicates whether to share the process namespace with the pod
	ShareProcessNamespace *bool `json:"shareProcessNamespace,omitempty"`
}

// SidecarConfigSpec defines the desired state of SidecarConfig
type SidecarConfigSpec struct {
	// Injection contains the configuration settings for the injection process.
	Injection InjectConfig `json:"injection"`
	// Kidecar contains the specific configuration settings for the Kidecar system.
	Kidecar KidecarConfig `json:"kidecar"`
}

type KidecarConfig struct {
	Plugins           []PluginConfig    `json:"plugins"`           // 启动的插件及其配置
	RestartPolicy     string            `json:"restartPolicy"`     // 重启策略
	Resources         map[string]string `json:"resources"`         // Sidecar 所需的资源
	SidecarStartOrder string            `json:"sidecarStartOrder"` // Sidecar 的启动顺序，是在主容器之后还是之前
}

// PluginConfig 表示插件的配置
type PluginConfig struct {
	Name string `json:"name"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Config    runtime.RawExtension `json:"config"`
	BootOrder int                  `json:"bootOrder"`
}

// SidecarConfigStatus defines the observed state of SidecarConfig
type SidecarConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// matchedPods is the number of Pods whose labels are matched with this SidecarSet's selector and are created after sidecarset creates
	MatchedPods int32 `json:"matchedPods"`

	// updatedPods is the number of matched Pods that are injected with the latest SidecarSet's containers
	UpdatedPods int32 `json:"updatedPods"`

	// readyPods is the number of matched Pods that have a ready condition
	ReadyPods int32 `json:"readyPods"`

	// updatedReadyPods is the number of matched pods that updated and ready
	UpdatedReadyPods int32 `json:"updatedReadyPods,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=scc
// +kubebuilder:printcolumn:name="MatchedPods",type="integer",JSONPath=".status.matchedPods"
// +kubebuilder:printcolumn:name="UpdatedPods",type="integer",JSONPath=".status.updatedPods"
// +kubebuilder:printcolumn:name="ReadyPods",type="integer",JSONPath=".status.readyPods"
// SidecarConfig is the Schema for the sidecarconfigs API
type SidecarConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SidecarConfigSpec   `json:"spec,omitempty"`
	Status SidecarConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// SidecarConfigList contains a list of SidecarConfig
type SidecarConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SidecarConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SidecarConfig{}, &SidecarConfigList{})
}
