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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	trusteev1alpha1 "github.com/confidential-containers/trustee-operator/api/v1alpha1"
)

var _ = Describe("TrusteeConfig Controller", func() {
	Context("When reconciling a Permissive TrusteeConfig", func() {
		const resourceName = "test-trusteeconfig"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: testNamespace,
		}

		BeforeEach(func() {
			tc := &trusteev1alpha1.TrusteeConfig{}
			err := k8sClient.Get(ctx, typeNamespacedName, tc)
			if err != nil && errors.IsNotFound(err) {
				resource := &trusteev1alpha1.TrusteeConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: testNamespace,
					},
					Spec: trusteev1alpha1.TrusteeConfigSpec{
						Profile: trusteev1alpha1.ProfilePermissive,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &trusteev1alpha1.TrusteeConfig{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should create the TrusteeConfig CR successfully", func() {
			tc := &trusteev1alpha1.TrusteeConfig{}
			err := k8sClient.Get(ctx, typeNamespacedName, tc)
			Expect(err).NotTo(HaveOccurred())
			Expect(tc.Spec.Profile).To(Equal(trusteev1alpha1.ProfilePermissive))
		})

		It("should build correct Trustee spec for Permissive profile", func() {
			reconciler := &TrusteeConfigReconciler{}
			tc := &trusteev1alpha1.TrusteeConfig{
				Spec: trusteev1alpha1.TrusteeConfigSpec{
					Profile: trusteev1alpha1.ProfilePermissive,
				},
			}

			spec := reconciler.buildTrusteeSpec(tc)
			Expect(spec.LogLevel).To(Equal("debug"))
		})

		It("should map NodePort service type", func() {
			reconciler := &TrusteeConfigReconciler{}
			tc := &trusteev1alpha1.TrusteeConfig{
				Spec: trusteev1alpha1.TrusteeConfigSpec{
					Profile:        trusteev1alpha1.ProfilePermissive,
					KbsServiceType: corev1.ServiceTypeNodePort,
				},
			}

			spec := reconciler.buildTrusteeSpec(tc)
			Expect(spec.NodePort.Enabled).To(BeTrue())
		})

		It("should propagate replicaCount to all components", func() {
			reconciler := &TrusteeConfigReconciler{}
			tc := &trusteev1alpha1.TrusteeConfig{
				Spec: trusteev1alpha1.TrusteeConfigSpec{
					Profile:      trusteev1alpha1.ProfilePermissive,
					ReplicaCount: 3,
				},
			}

			spec := reconciler.buildTrusteeSpec(tc)
			Expect(spec.KBS.ReplicaCount).To(Equal(int32(3)))
			Expect(spec.AS.ReplicaCount).To(Equal(int32(3)))
			Expect(spec.RVPS.ReplicaCount).To(Equal(int32(3)))
		})

		It("should map LoadBalancer service type", func() {
			reconciler := &TrusteeConfigReconciler{}
			tc := &trusteev1alpha1.TrusteeConfig{
				Spec: trusteev1alpha1.TrusteeConfigSpec{
					Profile:        trusteev1alpha1.ProfilePermissive,
					KbsServiceType: corev1.ServiceTypeLoadBalancer,
				},
			}

			spec := reconciler.buildTrusteeSpec(tc)
			Expect(spec.KBS.Service.ExposeLoadBalancer).To(BeTrue())
		})
	})
})
