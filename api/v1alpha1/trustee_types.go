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

// StorageBackendType defines the KV backend type.
// +kubebuilder:validation:Enum=LocalFs;LocalJson;Postgres;Memory
type StorageBackendType string

const (
	StorageBackendLocalFs   StorageBackendType = "LocalFs"
	StorageBackendLocalJson StorageBackendType = "LocalJson"
	StorageBackendPostgres  StorageBackendType = "Postgres"
	StorageBackendMemory    StorageBackendType = "Memory"
)

type LocalFsPersistence struct {
	// +optional
	KBS string `json:"kbs,omitempty"`
	// +optional
	AS string `json:"as,omitempty"`
	// +optional
	RVPS string `json:"rvps,omitempty"`
}

type LocalFsSpec struct {
	// +optional
	Persistence LocalFsPersistence `json:"persistence,omitempty"`
}

type LocalJsonSpec struct {
	// +optional
	Persistence LocalFsPersistence `json:"persistence,omitempty"`
}

type PostgresExternalSpec struct {
	// +optional
	ExistingSecretName string `json:"existingSecretName,omitempty"`
	// +optional
	ExistingSecretKey string `json:"existingSecretKey,omitempty"`
}

type PostgresInternalSpec struct {
	// +optional
	InitKvTables *bool `json:"initKvTables,omitempty"`
}

// +kubebuilder:validation:Enum=internal;external
type PostgresMode string

type PostgresSpec struct {
	// +kubebuilder:default=internal
	// +optional
	Mode PostgresMode `json:"mode,omitempty"`
	// +optional
	External PostgresExternalSpec `json:"external,omitempty"`
	// +optional
	Internal PostgresInternalSpec `json:"internal,omitempty"`
}

type StorageBackendSpec struct {
	// +kubebuilder:default=LocalFs
	// +optional
	Type StorageBackendType `json:"type,omitempty"`
	// +optional
	LocalFs LocalFsSpec `json:"localFs,omitempty"`
	// +optional
	LocalJson LocalJsonSpec `json:"localJson,omitempty"`
	// +optional
	Postgres PostgresSpec `json:"postgres,omitempty"`
}

type ImageSpec struct {
	// +optional
	Repository string `json:"repository,omitempty"`
	// +optional
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
	// +optional
	Tag string `json:"tag,omitempty"`
}

type ServiceSpec struct {
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`
	// +optional
	Port int32 `json:"port,omitempty"`
	// +optional
	LoadBalancerAnnotations map[string]string `json:"loadBalancerAnnotations,omitempty"`
}

type KBSAdminConfig struct {
	// +optional
	Issuer string `json:"issuer,omitempty"`
	// +optional
	Audience string `json:"audience,omitempty"`
	// +optional
	Role string `json:"role,omitempty"`
}

type AttestationServiceClientConfig struct {
	// +optional
	PoolSize int32 `json:"poolSize,omitempty"`
	// +optional
	Timeout int32 `json:"timeout,omitempty"`
}

type KBSConfig struct {
	// +optional
	Admin KBSAdminConfig `json:"admin,omitempty"`
	// +optional
	AttestationService AttestationServiceClientConfig `json:"attestationService,omitempty"`
}

type KBSServiceSpec struct {
	// +optional
	ExposeLoadBalancer bool `json:"exposeLoadBalancer,omitempty"`
	// +optional
	Port int32 `json:"port,omitempty"`
	// +optional
	LoadBalancerAnnotations map[string]string `json:"loadBalancerAnnotations,omitempty"`
}

type KBSSpec struct {
	// +kubebuilder:default=1
	// +optional
	ReplicaCount int32 `json:"replicaCount,omitempty"`
	// +optional
	Config KBSConfig `json:"config,omitempty"`
	// +optional
	Image ImageSpec `json:"image,omitempty"`
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// +optional
	Service KBSServiceSpec `json:"service,omitempty"`
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// +optional
	ExtraEnvVars []corev1.EnvVar `json:"extraEnvVars,omitempty"`
	// +optional
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`
	// +optional
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

type NvidiaVerifierSpec struct {
	// +kubebuilder:default=Local
	// +optional
	Type string `json:"type,omitempty"`
	// +optional
	VerifierURL string `json:"verifierUrl,omitempty"`
}

type DCAPVerifierSpec struct {
	// +optional
	CollateralService string `json:"collateral_service,omitempty"`
	// +optional
	TCBUpdateType string `json:"tcb_update_type,omitempty"`
}

type SEVerifierSpec struct {
	// +optional
	CredsDir string `json:"credsDir,omitempty"`
	// +optional
	NodeName string `json:"nodeName,omitempty"`
}

type VerifierSpec struct {
	// +optional
	Nvidia NvidiaVerifierSpec `json:"nvidia,omitempty"`
	// +optional
	DCAP DCAPVerifierSpec `json:"dcap,omitempty"`
	// +optional
	SE SEVerifierSpec `json:"se,omitempty"`
}

type ASSpec struct {
	// +kubebuilder:default=1
	// +optional
	ReplicaCount int32 `json:"replicaCount,omitempty"`
	// +optional
	Image ImageSpec `json:"image,omitempty"`
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// +optional
	Verifier VerifierSpec `json:"verifier,omitempty"`
	// +optional
	Service ServiceSpec `json:"service,omitempty"`
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// +optional
	ExtraEnvVars []corev1.EnvVar `json:"extraEnvVars,omitempty"`
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

type RVPSServiceSpec struct {
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`
	// +optional
	Port int32 `json:"port,omitempty"`
	// +optional
	LoadBalancerType string `json:"loadBalancerType,omitempty"`
	// +optional
	LoadBalancerAnnotations map[string]string `json:"loadBalancerAnnotations,omitempty"`
	// +optional
	PublicLoadBalancerAnnotations map[string]string `json:"publicLoadBalancerAnnotations,omitempty"`
}

type RVPSSpec struct {
	// +kubebuilder:default=1
	// +optional
	ReplicaCount int32 `json:"replicaCount,omitempty"`
	// +optional
	Image ImageSpec `json:"image,omitempty"`
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// +optional
	Service RVPSServiceSpec `json:"service,omitempty"`
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// +optional
	ExtraEnvVars []corev1.EnvVar `json:"extraEnvVars,omitempty"`
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

type SecretsSpec struct {
	// +kubebuilder:default=true
	// +optional
	UseEphemeralGeneratedKeys *bool `json:"useEphemeralGeneratedKeys,omitempty"`
	// +optional
	ExistingSecretName string `json:"existingSecretName,omitempty"`
}

type PostgreSQLAuthSpec struct {
	// +optional
	Username string `json:"username,omitempty"`
	// +optional
	Password string `json:"password,omitempty"`
	// +optional
	Database string `json:"database,omitempty"`
}

type PostgreSQLPersistenceSpec struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// +optional
	Size string `json:"size,omitempty"`
	// +optional
	StorageClass string `json:"storageClass,omitempty"`
	// +optional
	ExistingClaim string `json:"existingClaim,omitempty"`
}

type PostgreSQLPrimarySpec struct {
	// +optional
	Persistence PostgreSQLPersistenceSpec `json:"persistence,omitempty"`
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type PostgreSQLSpec struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// +optional
	NameOverride string `json:"nameOverride,omitempty"`
	// +optional
	Auth PostgreSQLAuthSpec `json:"auth,omitempty"`
	// +optional
	Primary PostgreSQLPrimarySpec `json:"primary,omitempty"`
}

type IngressTLSSpec struct {
	// +optional
	SecretName string `json:"secretName,omitempty"`
	// +optional
	Hosts []string `json:"hosts,omitempty"`
}

type IngressSpec struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// +optional
	ClassName string `json:"className,omitempty"`
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// +optional
	Host string `json:"host,omitempty"`
	// +optional
	TLS []IngressTLSSpec `json:"tls,omitempty"`
}

type NodePortSpec struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// +optional
	Port string `json:"port,omitempty"`
}

type BootstrapJobImageSpec struct {
	// +optional
	Repository string `json:"repository,omitempty"`
	// +optional
	Tag string `json:"tag,omitempty"`
	// +optional
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
}

type BootstrapUserKeysJobSpec struct {
	// +optional
	KeygenImage BootstrapJobImageSpec `json:"keygenImage,omitempty"`
	// +optional
	KubectlImage BootstrapJobImageSpec `json:"kubectlImage,omitempty"`
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// TrusteeSpec defines the desired state of a Trustee deployment.
type TrusteeSpec struct {
	// +optional
	NameOverride string `json:"nameOverride,omitempty"`
	// +optional
	FullnameOverride string `json:"fullnameOverride,omitempty"`

	// +kubebuilder:default=info
	// +kubebuilder:validation:Enum=info;debug;warn;error
	// +optional
	LogLevel string `json:"logLevel,omitempty"`

	// +optional
	DNSHostAliasWorkaround bool `json:"dnsHostAliasWorkaround,omitempty"`

	// +kubebuilder:default=Memory
	// +optional
	SessionStorageType string `json:"sessionStorageType,omitempty"`

	// +optional
	StorageBackend StorageBackendSpec `json:"storageBackend,omitempty"`

	// +optional
	KBS KBSSpec `json:"kbs,omitempty"`

	// +optional
	AS ASSpec `json:"as,omitempty"`

	// +optional
	RVPS RVPSSpec `json:"rvps,omitempty"`

	// +optional
	Secrets SecretsSpec `json:"secrets,omitempty"`

	// +optional
	PostgreSQL PostgreSQLSpec `json:"postgresql,omitempty"`

	// +optional
	Ingress IngressSpec `json:"ingress,omitempty"`

	// +optional
	NodePort NodePortSpec `json:"nodePort,omitempty"`

	// +optional
	BootstrapUserKeysJob BootstrapUserKeysJobSpec `json:"bootstrapUserKeysJob,omitempty"`
}

const (
	ConditionTypeHelmReleaseReady = "HelmReleaseReady"
	ConditionTypeKBSReady         = "KBSReady"
	ConditionTypeASReady          = "ASReady"
	ConditionTypeRVPSReady        = "RVPSReady"
)

// TrusteeStatus defines the observed state of Trustee.
type TrusteeStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`
	// +optional
	ReleaseRevision int `json:"releaseRevision,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Release",type=string,JSONPath=`.status.releaseName`
// +kubebuilder:printcolumn:name="Revision",type=integer,JSONPath=`.status.releaseRevision`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="HelmReleaseReady")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Trustee is the Schema for the trustees API.
type Trustee struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrusteeSpec   `json:"spec,omitempty"`
	Status TrusteeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TrusteeList contains a list of Trustee.
type TrusteeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Trustee `json:"items"`
}
