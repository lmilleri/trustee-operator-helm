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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	trusteev1alpha1 "github.com/confidential-containers/trustee-operator/api/v1alpha1"
	trusteehelm "github.com/confidential-containers/trustee-operator/internal/helm"
)

const (
	trusteeFinalizer = "trustee.confidentialcontainers.org/finalizer"
	requeueAfter     = 30 * time.Second
)

// TrusteeReconciler reconciles a Trustee object
type TrusteeReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	RESTGetter genericclioptions.RESTClientGetter
}

// +kubebuilder:rbac:groups=trustee.confidentialcontainers.org,resources=trustees,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=trustee.confidentialcontainers.org,resources=trustees/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=trustee.confidentialcontainers.org,resources=trustees/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services;configmaps;secrets;serviceaccounts;persistentvolumes;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings;clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

func (r *TrusteeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	trustee := &trusteev1alpha1.Trustee{}
	if err := r.Get(ctx, req.NamespacedName, trustee); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !trustee.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, trustee)
	}

	if !controllerutil.ContainsFinalizer(trustee, trusteeFinalizer) {
		controllerutil.AddFinalizer(trustee, trusteeFinalizer)
		if err := r.Update(ctx, trustee); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	helmClient := trusteehelm.NewClient(r.RESTGetter, trustee.Namespace)
	releaseName := r.releaseName(trustee)

	chartVersion, err := trusteehelm.ChartVersion()
	if err != nil {
		logger.Error(err, "failed to read chart version")
		return ctrl.Result{}, err
	}

	needsHelmSync := trustee.Generation != trustee.Status.ObservedGeneration ||
		trustee.Status.ChartVersion != chartVersion

	if needsHelmSync {
		vals, err := trusteehelm.SpecToValues(&trustee.Spec)
		if err != nil {
			logger.Error(err, "failed to convert spec to Helm values")
			r.setCondition(ctx, trustee, trusteev1alpha1.ConditionTypeHelmReleaseReady,
				metav1.ConditionFalse, "SpecConversionFailed", err.Error())
			return ctrl.Result{}, err
		}

		rel, err := helmClient.InstallOrUpgrade(releaseName, vals)
		if err != nil {
			logger.Error(err, "failed to install/upgrade Helm release")
			r.setCondition(ctx, trustee, trusteev1alpha1.ConditionTypeHelmReleaseReady,
				metav1.ConditionFalse, "HelmReleaseFailed", err.Error())
			return ctrl.Result{RequeueAfter: requeueAfter}, err
		}

		trustee.Status.ReleaseName = rel.Name
		trustee.Status.ReleaseRevision = rel.Version
		trustee.Status.ObservedGeneration = trustee.Generation
		trustee.Status.ChartVersion = chartVersion

		r.setCondition(ctx, trustee, trusteev1alpha1.ConditionTypeHelmReleaseReady,
			metav1.ConditionTrue, "HelmReleaseReady",
			fmt.Sprintf("Release %s at revision %d", rel.Name, rel.Version))
	}

	r.updateComponentStatus(ctx, trustee)

	if err := r.Status().Update(ctx, trustee); err != nil {
		logger.Error(err, "failed to update Trustee status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

func (r *TrusteeReconciler) handleDeletion(ctx context.Context, trustee *trusteev1alpha1.Trustee) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	if controllerutil.ContainsFinalizer(trustee, trusteeFinalizer) {
		helmClient := trusteehelm.NewClient(r.RESTGetter, trustee.Namespace)
		releaseName := r.releaseName(trustee)

		if err := helmClient.Uninstall(releaseName); err != nil {
			logger.Error(err, "failed to uninstall Helm release")
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(trustee, trusteeFinalizer)
		if err := r.Update(ctx, trustee); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *TrusteeReconciler) releaseName(trustee *trusteev1alpha1.Trustee) string {
	if trustee.Spec.FullnameOverride != "" {
		return trustee.Spec.FullnameOverride
	}
	return trustee.Name
}

func (r *TrusteeReconciler) updateComponentStatus(ctx context.Context, trustee *trusteev1alpha1.Trustee) {
	releaseName := r.releaseName(trustee)

	r.checkDeploymentReady(ctx, trustee, releaseName+"-kbs",
		trusteev1alpha1.ConditionTypeKBSReady, "KBS")
	r.checkDeploymentReady(ctx, trustee, releaseName+"-as",
		trusteev1alpha1.ConditionTypeASReady, "AS")
	r.checkDeploymentReady(ctx, trustee, releaseName+"-rvps",
		trusteev1alpha1.ConditionTypeRVPSReady, "RVPS")
}

func (r *TrusteeReconciler) checkDeploymentReady(ctx context.Context, trustee *trusteev1alpha1.Trustee,
	deployName string, conditionType string, component string) {

	deploy := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      deployName,
		Namespace: trustee.Namespace,
	}, deploy)

	if err != nil {
		r.setCondition(ctx, trustee, conditionType,
			metav1.ConditionFalse, "DeploymentNotFound",
			fmt.Sprintf("%s deployment %s not found", component, deployName))
		return
	}

	if deploy.Status.ReadyReplicas == deploy.Status.Replicas && deploy.Status.Replicas > 0 {
		r.setCondition(ctx, trustee, conditionType,
			metav1.ConditionTrue, component+"Ready",
			fmt.Sprintf("%s has %d/%d replicas ready", component, deploy.Status.ReadyReplicas, deploy.Status.Replicas))
	} else {
		r.setCondition(ctx, trustee, conditionType,
			metav1.ConditionFalse, component+"NotReady",
			fmt.Sprintf("%s has %d/%d replicas ready", component, deploy.Status.ReadyReplicas, deploy.Status.Replicas))
	}
}

func (r *TrusteeReconciler) setCondition(_ context.Context, trustee *trusteev1alpha1.Trustee,
	conditionType string, status metav1.ConditionStatus, reason, message string) {

	meta.SetStatusCondition(&trustee.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: trustee.Generation,
		Reason:             reason,
		Message:            message,
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *TrusteeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&trusteev1alpha1.Trustee{}).
		Owns(&appsv1.Deployment{}).
		Named("trustee").
		Complete(r)
}
