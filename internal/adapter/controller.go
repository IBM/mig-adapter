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
	"strings"

	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	CONDITION_MESSAGE_SEPARATOR = ","
)

var aclog = logf.Log.WithName("adapter controller")

func (a *Adapter) CheckAndRestorePodsWithContext(ctx context.Context, nodes []corev1.Node, podItems []corev1.Pod) []*corev1.Pod {

	available := a.detectAllAvailableMIGs(nodes, podItems)
	if len(available) == 0 {
		return nil
	}
	order := a.buildOrderedMIGList(available)

	podsToRestart := []*corev1.Pod{}

	// start with the larget mig demand for best gain
	pods := a.filterAndSortPodsDescendingByMIG(podItems)
	for _, pod := range pods {
		org, exists := pod.Annotations[ADAPTER_ANNOTATION_PREFIX+ADAPTER_ANNOTATION_ORIGINAL]
		if !exists {
			continue
		}

		records := PodResources{}
		err := json.Unmarshal([]byte(org), &records)
		if err != nil || len(records) == 0 {
			continue
		}

		restart := false
		for _, c := range pod.Spec.Containers {
			original := records[c.Name]
			updated := a.checkAndSizeUpMIGForContainerResource(original.Requests, original.Limits, pod.Spec.NodeSelector, available, order)
			if a.compareMIGResources(original.Requests, c.Resources.Requests) == -1 {
				restart = true
				if updated {
					a.storeResourceRulesForContainer(a.genPodKey(pod), c.Name, original.Requests, original.Limits, original.Claims)
				}
				aclog.Info("controller restore", "original", records[c.Name], "updated", original)
			}
		}

		if restart {
			podsToRestart = append(podsToRestart, pod.DeepCopy())
		}
	}

	return podsToRestart
}

func (a *Adapter) IsPodPendingForMIGs(pod *corev1.Pod) bool {

	return a.PodPendingForMIG(pod) != ""
}

func (a *Adapter) PodPendingForMIG(pod *corev1.Pod) corev1.ResourceName {

	mig := corev1.ResourceName("")

	if pod.Status.Phase != corev1.PodPending {
		return mig
	}

	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse && cond.Reason == corev1.PodReasonUnschedulable {
			start := strings.Index(cond.Message, PODMESSAGE_INSUFFICIENT_MIG_PREFIX)
			if start != -1 {
				msg := cond.Message[start:]
				end := strings.Index(msg, CONDITION_MESSAGE_SEPARATOR)
				mig = corev1.ResourceName(msg[strings.Index(msg, RESOURCE_MIG_PREFIX):end])
				break
			}
		}
	}

	return mig
}

func (a *Adapter) AdaptPodToGPUsWithContext(ctx context.Context, pod *corev1.Pod, nodes []corev1.Node, pods []corev1.Pod) bool {

	aclog.Info("pod pending mig", "name", pod.Name, "namespace", pod.Namespace)

	restart := false
	podkey := a.genPodKey(pod)

	available, order := a.getAvailableMIGsAndOrder(nodes, pods)
	if len(available) == 0 || len(order) == 0 {
		return false
	}

	for _, c := range pod.Spec.Containers {
		req := c.Resources.Requests
		limits := c.Resources.Limits
		claims := c.Resources.Claims

		if a.checkAndSizeUpMIGForContainerResource(req, limits, pod.Spec.NodeSelector, available, order) {
			restart = true
			a.storeResourceRulesForContainer(podkey, c.Name, req, limits, claims)
		}
	}

	return restart
}

func (a *Adapter) AdaptGPUsToPodWithContext(ctx context.Context, pod *corev1.Pod, nodes []corev1.Node, pods []corev1.Pod) *corev1.Node {

	mig := a.PodPendingForMIG(pod)
	if mig == "" {
		return nil
	}

	profile := a.buildMIGProfileMap(nil)

	available, _ := a.getAvailableMIGsAndOrder(nodes, pods)

	node := a.findAvailableNodeWithFreeGPU(pod.Spec.NodeSelector, nodes, available)
	if node != nil {
		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}
		node.Labels[LABELKEY_MIG_CONFIG] = profile[mig]

	}

	return node
}
