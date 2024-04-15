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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("API for Adapter", func() {

	adapter := GetAdapter(cli)

	Context("For given Pod without rules", func() {
		It("should do nothing", func() {
			pod := _test_pod1.DeepCopy()

			adapter.CheckAndUpdatePodWithContext(ctx, pod)
			Expect(pod.Annotations).To(BeNil())
			Expect(pod.Spec.Containers[0].Resources).To(BeEquivalentTo(_test_pod1.Spec.Containers[0].Resources))
		})
	})

	Context("For given Pod with rules", func() {
		It("should be patched correctly", func() {
			pod := _test_pod1.DeepCopy()
			podkey := adapter.genPodKey(pod)
			creq := corev1.ResourceList{
				_test_mig_Identifier_string_2_10: _test_quantity_1,
			}

			climit := corev1.ResourceList{
				_test_mig_Identifier_string_2_10: _test_quantity_1,
			}
			cclaims := []corev1.ResourceClaim{
				{
					Name: "testclaim",
				},
			}

			adapter.storeResourceRulesForContainer(podkey, _test_container1_name, creq, climit, cclaims)
			adapter.CheckAndUpdatePodWithContext(ctx, pod)
			Expect(pod.Spec.Containers[0].Resources.Requests).To(BeEquivalentTo(creq))
			Expect(pod.Spec.Containers[0].Resources.Limits).To(BeEquivalentTo(climit))
			Expect(pod.Spec.Containers[0].Resources.Claims).To(BeEquivalentTo(cclaims))
			Expect(pod.Annotations).NotTo(BeNil())
			org := pod.Annotations[ADAPTER_ANNOTATION_PREFIX+ADAPTER_ANNOTATION_ORIGINAL]
			Expect(org).NotTo(BeEmpty())

			pr := PodResources{}
			err := json.Unmarshal([]byte(org), &pr)
			Expect(err).NotTo(HaveOccurred())
			Expect(pr[_test_container1_name]).To(BeEquivalentTo(_test_pod1.Spec.Containers[0].Resources))
		})
	})
})
