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

package controller

import (
	"context"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	trusteev1alpha1 "github.com/confidential-containers/trustee-operator/api/v1alpha1"
)

// TrusteeConfigReconciler reconciles a TrusteeConfig object
type TrusteeConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=trustee.confidentialcontainers.org,resources=trusteeconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=trustee.confidentialcontainers.org,resources=trusteeconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=trustee.confidentialcontainers.org,resources=trusteeconfigs/finalizers,verbs=update

func (r *TrusteeConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	tc := &trusteev1alpha1.TrusteeConfig{}
	if err := r.Get(ctx, req.NamespacedName, tc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	desiredSpec := r.buildTrusteeSpec(tc)

	trustee := &trusteev1alpha1.Trustee{}
	trusteeName := types.NamespacedName{
		Name:      tc.Name,
		Namespace: tc.Namespace,
	}

	err := r.Get(ctx, trusteeName, trustee)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		trustee = &trusteev1alpha1.Trustee{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tc.Name,
				Namespace: tc.Namespace,
			},
			Spec: desiredSpec,
		}
		if err := controllerutil.SetControllerReference(tc, trustee, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("setting owner reference: %w", err)
		}
		if err := r.Create(ctx, trustee); err != nil {
			logger.Error(err, "failed to create Trustee")
			r.setCondition(tc, metav1.ConditionFalse, "TrusteeCreationFailed", err.Error())
			_ = r.Status().Update(ctx, tc)
			return ctrl.Result{}, err
		}
		logger.Info("created Trustee", "name", trustee.Name)
	} else {
		if !reflect.DeepEqual(trustee.Spec, desiredSpec) {
			trustee.Spec = desiredSpec
			if err := r.Update(ctx, trustee); err != nil {
				logger.Error(err, "failed to update Trustee")
				r.setCondition(tc, metav1.ConditionFalse, "TrusteeUpdateFailed", err.Error())
				_ = r.Status().Update(ctx, tc)
				return ctrl.Result{}, err
			}
			logger.Info("updated Trustee", "name", trustee.Name)
		}
	}

	tc.Status.ObservedGeneration = tc.Generation
	tc.Status.TrusteeRef = &corev1.ObjectReference{
		Kind:       "Trustee",
		APIVersion: trusteev1alpha1.GroupVersion.String(),
		Name:       trustee.Name,
		Namespace:  trustee.Namespace,
	}

	r.propagateStatus(tc, trustee)

	if err := r.Status().Update(ctx, tc); err != nil {
		logger.Error(err, "failed to update TrusteeConfig status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TrusteeConfigReconciler) buildTrusteeSpec(tc *trusteev1alpha1.TrusteeConfig) trusteev1alpha1.TrusteeSpec {
	replicas := tc.Spec.ReplicaCount
	spec := trusteev1alpha1.TrusteeSpec{
		LogLevel: "debug",
		KBS:      trusteev1alpha1.KBSSpec{ReplicaCount: replicas},
		AS:       trusteev1alpha1.ASSpec{ReplicaCount: replicas},
		RVPS:     trusteev1alpha1.RVPSSpec{ReplicaCount: replicas},
	}

	switch tc.Spec.KbsServiceType {
	case corev1.ServiceTypeNodePort:
		spec.NodePort = trusteev1alpha1.NodePortSpec{
			Enabled: true,
		}
	case corev1.ServiceTypeLoadBalancer:
		spec.KBS.Service.ExposeLoadBalancer = true
	}

	return spec
}

func (r *TrusteeConfigReconciler) propagateStatus(tc *trusteev1alpha1.TrusteeConfig, trustee *trusteev1alpha1.Trustee) {
	helmReady := meta.FindStatusCondition(trustee.Status.Conditions, trusteev1alpha1.ConditionTypeHelmReleaseReady)
	if helmReady != nil && helmReady.Status == metav1.ConditionTrue {
		r.setCondition(tc, metav1.ConditionTrue, "TrusteeReady",
			fmt.Sprintf("Trustee %s is ready", trustee.Name))
	} else {
		msg := "Trustee is not ready"
		if helmReady != nil {
			msg = helmReady.Message
		}
		r.setCondition(tc, metav1.ConditionFalse, "TrusteeNotReady", msg)
	}
}

func (r *TrusteeConfigReconciler) setCondition(tc *trusteev1alpha1.TrusteeConfig,
	status metav1.ConditionStatus, reason, message string) {

	meta.SetStatusCondition(&tc.Status.Conditions, metav1.Condition{
		Type:               trusteev1alpha1.ConditionTypeTrusteeConfigReady,
		Status:             status,
		ObservedGeneration: tc.Generation,
		Reason:             reason,
		Message:            message,
	})
}

func (r *TrusteeConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&trusteev1alpha1.TrusteeConfig{}).
		Owns(&trusteev1alpha1.Trustee{}).
		Named("trusteeconfig").
		Complete(r)
}
