/*
Copyright 2026.

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
)

// +kubebuilder:validation:Enum=Permissive
type ProfileType string

const (
	ProfilePermissive ProfileType = "Permissive"
)

// TrusteeConfigSpec defines the desired state of TrusteeConfig.
type TrusteeConfigSpec struct {
	// +kubebuilder:default=Permissive
	Profile ProfileType `json:"profile"`

	// +kubebuilder:default=1
	// +optional
	ReplicaCount int32 `json:"replicaCount,omitempty"`

	// +optional
	KbsServiceType corev1.ServiceType `json:"kbsServiceType,omitempty"`
}

// TrusteeConfigStatus defines the observed state of TrusteeConfig.
type TrusteeConfigStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +optional
	TrusteeRef *corev1.ObjectReference `json:"trusteeRef,omitempty"`
}

const (
	ConditionTypeTrusteeConfigReady = "Ready"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Profile",type=string,JSONPath=`.spec.profile`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TrusteeConfig is the Schema for the trusteeconfigs API.
type TrusteeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrusteeConfigSpec   `json:"spec,omitempty"`
	Status TrusteeConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TrusteeConfigList contains a list of TrusteeConfig.
type TrusteeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrusteeConfig `json:"items"`
}
