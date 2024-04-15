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
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

const (
	_test_mig_Identifier_string_1_5 = "nvidia.com/mig-1g.5gb"
	_test_mig_Identifier_compute    = 1
	_test_mig_Identifier_memory     = 5

	_test_mig_Identifier_string_2_10 = "nvidia.com/mig-2g.10gb"
	_test_mig_Identifier_string_3_20 = "nvidia.com/mig-3g.20gb"
	_test_mig_Identifier_string_4_20 = "nvidia.com/mig-4g.20gb"
)

var _ = Describe("MIG Unit Test", func() {
	var err error
	md := &migIdentifier{}

	Context("For MIG Identifier", func() {
		It("should be able to parse formatted string", func() {
			err = md.Parse(_test_mig_Identifier_string_1_5)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(md.Compute).To(Equal(_test_mig_Identifier_compute))
			Expect(md.Memory).To(Equal(_test_mig_Identifier_memory))
		})

		It("convert itself to string matching the format", func() {
			Expect(md.String()).Should(Equal(_test_mig_Identifier_string_1_5))
		})
	})

	Context("For MIG Identifier Array", func() {
		It("should be able to work with sort.Sort ", func() {
			m1 := &migIdentifier{}
			m1.Parse(_test_mig_Identifier_string_1_5)
			m2 := &migIdentifier{}
			m2.Parse(_test_mig_Identifier_string_2_10)
			m3 := &migIdentifier{}
			m3.Parse(_test_mig_Identifier_string_3_20)
			m4 := &migIdentifier{}
			m4.Parse(_test_mig_Identifier_string_4_20)

			org := OrderedmigIdentifierList{
				*m4, *m3, *m2, *m1,
			}
			sorted := OrderedmigIdentifierList{
				*m1, *m2, *m3, *m4,
			}
			sort.Sort(org)

			Expect(org).Should(BeEquivalentTo(sorted))
		})
	})

	Context("For Pod Identifier Array", func() {
		It("should be able to work with sort.Sort ", func() {
			pod1 := _test_pod1.DeepCopy()
			pod2 := _test_pod2.DeepCopy()
			pod2.Spec.Containers[0].Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					_test_mig_Identifier_string_2_10: _test_quantity_1,
				},
				Limits: corev1.ResourceList{
					_test_mig_Identifier_string_2_10: _test_quantity_1,
				},
			}
			pod3 := _test_podpending.DeepCopy()
			pod3.Spec.Containers[0].Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					_test_mig_Identifier_string_3_20: _test_quantity_1,
				},
				Limits: corev1.ResourceList{
					_test_mig_Identifier_string_3_20: _test_quantity_1,
				},
			}

			podItems := []corev1.Pod{*pod1, *pod3, *pod2}

			adapter := GetAdapter(cli)
			pods := adapter.filterAndSortPodsDescendingByMIG(podItems)
			Expect(pods).NotTo(BeEmpty())
			Expect(pods[0].Name).To(Equal(pod3.Name))
			Expect(pods[1].GenerateName).To(Equal(pod2.GenerateName))
			Expect(pods[2].Name).To(Equal(pod1.Name))
		})
	})

	Context("For a given node", func() {
		It("should be able to generate available mig map and ordered mig type list", func() {
			nodes := []corev1.Node{_test_node1, _test_node2}
			pods := []corev1.Pod{_test_pod1, _test_pod2}

			adapter := GetAdapter(cli)
			available, order := adapter.getAvailableMIGsAndOrder(nodes, pods)
			Expect(available).NotTo(BeNil())
			Expect(order).NotTo(BeNil())

		})
	})

})
