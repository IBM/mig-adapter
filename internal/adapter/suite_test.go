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
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	gpuv1alpha1 "github.com/IBM/mig-adapter/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var cli client.Client
var ctx = context.TODO()

var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.29.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = gpuv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	cli, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(cli).NotTo(BeNil())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var (
	_test_quantity_1 = resource.MustParse("1")
	_test_quantity_2 = resource.MustParse("2")

	_test_namespace = "test"

	_test_node1_name = "node1"
	_test_node2_name = "node2"

	_test_pod1_name    = "pod1"
	_test_pod2_genname = "pod2"

	_test_podpending_name = "podpending"

	_test_container1_name = "container1"

	_test_pod_status_condition_message_prefix = "0/2 nodes are available, " + PODMESSAGE_INSUFFICIENT_PREFIX
)

var (
	_test_node1 = corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: _test_node1_name,
		},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				_test_mig_Identifier_string_2_10: _test_quantity_1,
				_test_mig_Identifier_string_3_20: _test_quantity_1,
			},
		},
	}
	_test_node2 = corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: _test_node2_name,
		},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				_test_mig_Identifier_string_1_5:  _test_quantity_2,
				_test_mig_Identifier_string_2_10: _test_quantity_1,
				_test_mig_Identifier_string_3_20: _test_quantity_1,
			},
		},
	}

	_test_pod1 = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      _test_pod1_name,
			Namespace: _test_namespace,
		},
		Spec: corev1.PodSpec{
			NodeName: _test_node1_name,
			Containers: []corev1.Container{
				{
					Name: _test_container1_name,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							_test_mig_Identifier_string_1_5: _test_quantity_1,
						},
						Limits: corev1.ResourceList{
							_test_mig_Identifier_string_1_5: _test_quantity_1,
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	_test_pod2 = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: _test_pod2_genname,
			Namespace:    _test_namespace,
		},
		Spec: corev1.PodSpec{
			NodeName: _test_node2_name,
			Containers: []corev1.Container{
				{
					Name: _test_container1_name,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							_test_mig_Identifier_string_1_5: _test_quantity_1,
						},
						Limits: corev1.ResourceList{
							_test_mig_Identifier_string_1_5: _test_quantity_1,
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	_test_podpending = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: _test_podpending_name,
		},
		Spec: corev1.PodSpec{
			NodeName: _test_node2_name,
			Containers: []corev1.Container{
				{
					Name: _test_container1_name,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							_test_mig_Identifier_string_1_5: _test_quantity_1,
						},
						Limits: corev1.ResourceList{
							_test_mig_Identifier_string_1_5: _test_quantity_1,
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{
				{
					Type:    corev1.PodScheduled,
					Status:  corev1.ConditionFalse,
					Reason:  corev1.PodReasonUnschedulable,
					Message: _test_pod_status_condition_message_prefix + _test_mig_Identifier_string_1_5 + CONDITION_MESSAGE_SEPARATOR,
				},
			},
		},
	}
)
