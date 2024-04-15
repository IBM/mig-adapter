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

var _ = Describe("API for Controller", func() {

	adapter := GetAdapter(cli)

	Context("For a Pod in Pending status", func() {

		It("should be able to tell if it is pending for MIG resource or not", func() {
			pod := &corev1.Pod{}
			Expect(adapter.IsPodPendingForMIGs(pod)).To(BeFalse())
		})

		It("should not do anything if no MIG resource available", func() {
			var nodes []corev1.Node
			var pods []corev1.Pod
			pod := _test_podpending.DeepCopy()
			Expect(adapter.AdaptPodToGPUsWithContext(ctx, pod, nodes, pods)).To(BeFalse())
		})

		It("should be able to size it up, if larger MIG resource is available", func() {
			var nodes []corev1.Node
			var pods []corev1.Pod
			nodes = []corev1.Node{_test_node1}
			pod := _test_podpending.DeepCopy()

			targetMIG := corev1.ResourceList{
				_test_mig_Identifier_string_2_10: _test_quantity_1,
			}

			Expect(adapter.AdaptPodToGPUsWithContext(ctx, pod, nodes, pods)).To(BeTrue())
			Expect(pod.Spec.Containers[0].Resources.Requests).To(BeEquivalentTo(targetMIG))
		})

		It("should be able to restore pod when original MIG request is available", func() {
			pod := _test_pod2.DeepCopy()
			original := make(PodResources)
			container_resource := pod.Spec.Containers[0].Resources.DeepCopy()
			original[pod.Spec.Containers[0].Name] = *container_resource
			bytes, err := json.Marshal(original)
			Expect(err).NotTo(HaveOccurred())
			pod.Annotations = make(map[string]string)
			pod.Annotations[ADAPTER_ANNOTATION_PREFIX+ADAPTER_ANNOTATION_ORIGINAL] = string(bytes)

			pod.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
				_test_mig_Identifier_string_2_10: _test_quantity_1,
			}
			pod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
				_test_mig_Identifier_string_2_10: _test_quantity_1,
			}

			node := _test_node2.DeepCopy()
			nodes := []corev1.Node{*node}
			pods := []corev1.Pod{*pod}

			restored := adapter.CheckAndRestorePodsWithContext(ctx, nodes, pods)
			Expect(restored).NotTo(BeEmpty())
			Expect(restored[0].Name).To(Equal(pod.Name))
		})

		It("should be able to identify free node with GPU when applicable", func() {
			node1 := _test_node1.DeepCopy()
			node2 := _test_node2.DeepCopy()

			pod1 := _test_pod1.DeepCopy()
			pod2 := _test_pod2.DeepCopy()
			pod2.Spec.NodeName = _test_node1_name

			nodes := []corev1.Node{*node1, *node2}
			pods := []corev1.Pod{*pod1, *pod2}

			pod := _test_podpending.DeepCopy()
			pod.Status.Conditions[0].Message = _test_pod_status_condition_message_prefix + _test_mig_Identifier_string_4_20 + CONDITION_MESSAGE_SEPARATOR

			node := adapter.AdaptGPUsToPodWithContext(ctx, pod, nodes, pods)
			Expect(node.Name).To(Equal(node2.Name))
			Expect(node.Labels[LABELKEY_MIG_CONFIG]).To(Equal("all-4g.20gb"))
		})
	})
})
