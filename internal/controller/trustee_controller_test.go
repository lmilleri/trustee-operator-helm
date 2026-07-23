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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	trusteev1alpha1 "github.com/confidential-containers/trustee-operator/api/v1alpha1"
)

var _ = Describe("Trustee Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: testNamespace,
		}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Trustee")
			trustee := &trusteev1alpha1.Trustee{}
			err := k8sClient.Get(ctx, typeNamespacedName, trustee)
			if err != nil && errors.IsNotFound(err) {
				resource := &trusteev1alpha1.Trustee{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: testNamespace,
					},
					Spec: trusteev1alpha1.TrusteeSpec{
						LogLevel: "info",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &trusteev1alpha1.Trustee{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Trustee")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should create the CR successfully", func() {
			trustee := &trusteev1alpha1.Trustee{}
			err := k8sClient.Get(ctx, typeNamespacedName, trustee)
			Expect(err).NotTo(HaveOccurred())
			Expect(trustee.Spec.LogLevel).To(Equal("info"))
		})
	})
})
