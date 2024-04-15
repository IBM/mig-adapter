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

package webhook

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	gpuadapter "github.com/IBM/mig-adapter/internal/adapter"
)

// log is for logging in this package.
var whlog = logf.Log.WithName("webhook")

type PodDefaulter struct {
	Adapter *gpuadapter.Adapter
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (pd *PodDefaulter) SetupWebhookWithManager(mgr ctrl.Manager) error {

	return ctrl.NewWebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(pd).
		Complete()
}

func (pd *PodDefaulter) Default(ctx context.Context, obj runtime.Object) error {

	pod := obj.(*corev1.Pod)

	whlog.Info("defaulter", "pod name", pod.Name, "pod genname", pod.GenerateName)
	pd.Adapter.CheckAndUpdatePodWithContext(context.TODO(), pod)

	return nil
}
