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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gpuadapter "github.com/IBM/mig-adapter/internal/adapter"
)

var clog = logf.Log.WithName("pod controller")

type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Adapter *gpuadapter.Adapter
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}

//+kubebuilder:rbac:resources=pods,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:resources=pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NVidiaMIGAdapter object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			// Deleted
			nodes, pods := r.GetAllNodesAndPodsWithContext(ctx)
			podsToRestore := r.Adapter.CheckAndRestorePodsWithContext(ctx, nodes, pods)
			for _, pod := range podsToRestore {
				r.Delete(ctx, pod)
			}
			clog.Info("reconciler", "deleted", req.NamespacedName.String())

			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if r.Adapter.IsPodPendingForMIGs(pod) {
		nodes, pods := r.GetAllNodesAndPodsWithContext(ctx)
		restart := r.Adapter.AdaptPodToGPUsWithContext(ctx, pod, nodes, pods)

		if restart {
			r.Delete(ctx, pod)
		} else {
			node := r.Adapter.AdaptGPUsToPodWithContext(ctx, pod, nodes, pods)
			if node != nil {
				r.Update(ctx, node, &client.UpdateOptions{})
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *PodReconciler) GetAllNodesAndPodsWithContext(ctx context.Context) ([]corev1.Node, []corev1.Pod) {
	nodelist := &corev1.NodeList{}
	err := r.List(ctx, nodelist)
	if err != nil {
		clog.Error(err, "load allocable from all nodes")
		return nil, nil
	}

	podlist := &corev1.PodList{}
	err = r.List(ctx, podlist)
	if err != nil {
		clog.Error(err, "load all pods")
		return nil, nil
	}

	return nodelist.Items, podlist.Items
}
