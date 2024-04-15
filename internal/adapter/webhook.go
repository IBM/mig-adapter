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
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var awlog = logf.Log.WithName("adapter controller")

func (a *Adapter) CheckAndUpdatePodWithContext(ctx context.Context, pod *corev1.Pod) {

	podkey := a.genPodKey(pod)

	original := make(PodResources)

	for i, c := range pod.Spec.Containers {
		req, limits, claims := a.getResourceRulesForContainer(podkey, c.Name)
		container_original := corev1.ResourceRequirements{}
		done := false

		if req != nil {
			container_original.Requests = pod.Spec.Containers[i].Resources.Requests
			done = true

			pod.Spec.Containers[i].Resources.Requests = req
			awlog.Info("update pod res requests", "container name", pod.Spec.Containers[0].Name, "update req", req)
		}

		if limits != nil {
			container_original.Limits = pod.Spec.Containers[i].Resources.Limits
			done = true
			pod.Spec.Containers[i].Resources.Limits = limits
			awlog.Info("update pod res limits", "container name", pod.Spec.Containers[0].Name, "update limits", limits)
		}

		if claims != nil {
			done = true
			container_original.Claims = pod.Spec.Containers[i].Resources.Claims

			pod.Spec.Containers[i].Resources.Claims = claims
			awlog.Info("update pod res claims", "container name", pod.Spec.Containers[0].Name, "update claims", claims)
		}

		if done {
			original[c.Name] = container_original
		}
	}

	if len(original) > 0 {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		bytes, err := json.Marshal(original)
		if err == nil {
			pod.Annotations[ADAPTER_ANNOTATION_PREFIX+ADAPTER_ANNOTATION_ORIGINAL] = string(bytes)
		}
	}

	a.removeResourceRulesForPod(podkey)
}
