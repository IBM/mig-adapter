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

package adapter

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("API for Adapter", func() {

	var adapter *Adapter

	Context("For initialization", func() {
		It("adapter should be able to initialize", func() {
			adapter = GetAdapter(cli)
			Expect(adapter).NotTo(BeNil())
		})

		It("should be singleton", func() {
			another := GetAdapter(cli)
			Expect(another).To(Equal(adapter))
		})

	})

	Context("For a given Pod", func() {
		It("should be able to generate podkey for pod", func() {
			pod := _test_pod1.DeepCopy()
			podkey := adapter.genPodKey(pod)
			Expect(podkey).To(Equal(types.NamespacedName{
				Namespace: _test_namespace,
				Name:      _test_pod1_name,
			}))

			pod = _test_pod2.DeepCopy()
			podkey = adapter.genPodKey(pod)
			Expect(podkey).To(Equal(types.NamespacedName{
				Namespace: _test_namespace,
				Name:      _test_pod2_genname,
			}))
		})

	})

	Context("For given rules", func() {
		It("should be stored, retrieved, deleted correctly", func() {
			pod := _test_pod1.DeepCopy()
			podkey := adapter.genPodKey(pod)
			req, limit, claims := adapter.getResourceRulesForContainer(podkey, _test_container1_name)
			Expect(req).To(BeNil())
			Expect(limit).To(BeNil())
			Expect(claims).To(BeNil())

			creq := corev1.ResourceList{
				_test_mig_Identifier_string_1_5: _test_quantity_1,
			}

			climit := corev1.ResourceList{
				_test_mig_Identifier_string_1_5: _test_quantity_1,
			}
			cclaims := []corev1.ResourceClaim{
				{
					Name: "testclaim",
				},
			}

			adapter.storeResourceRulesForContainer(podkey, _test_container1_name, creq, climit, cclaims)

			req, limit, claims = adapter.getResourceRulesForContainer(podkey, _test_container1_name)
			Expect(req).To(BeEquivalentTo(creq))
			Expect(limit).To(BeEquivalentTo(climit))
			Expect(claims).To(BeEquivalentTo(cclaims))

			adapter.removeResourceRulesForPod(podkey)
			req, limit, claims = adapter.getResourceRulesForContainer(podkey, _test_container1_name)
			Expect(req).To(BeNil())
			Expect(limit).To(BeNil())
			Expect(claims).To(BeNil())

		})
	})
})
