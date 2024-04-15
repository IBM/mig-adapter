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
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ADAPTER_ANNOTATION_PREFIX   = "adapter.gpu.turbonomic.ibm.com/"
	ADAPTER_ANNOTATION_ORIGINAL = "original"
)

type PodResources map[string]corev1.ResourceRequirements

type Adapter struct {
	client.Client

	m     sync.RWMutex
	rules map[types.NamespacedName]PodResources
}

var _adapter *Adapter

func GetAdapter(cli client.Client) *Adapter {

	if _adapter == nil {
		_adapter = &Adapter{
			Client: cli,
		}
	}

	if _adapter.rules == nil {
		_adapter.rules = make(map[types.NamespacedName]PodResources)
	}

	return _adapter
}

func (a *Adapter) genPodKey(pod *corev1.Pod) types.NamespacedName {
	podkey := types.NamespacedName{
		Namespace: pod.Namespace,
	}

	if pod.GenerateName != "" {
		podkey.Name = pod.GenerateName
	} else {
		podkey.Name = pod.Name
	}
	return podkey
}

func (a *Adapter) removeResourceRulesForPod(podkey types.NamespacedName) {
	a.m.Lock()
	defer a.m.Unlock()

	delete(a.rules, podkey)
}

func (a *Adapter) storeResourceRulesForContainer(podkey types.NamespacedName, container string, req corev1.ResourceList, limits corev1.ResourceList, claims []corev1.ResourceClaim) {
	a.m.Lock()
	defer a.m.Unlock()

	podRes, exists := a.rules[podkey]

	if !exists {
		podRes = make(PodResources)
	}

	containerRes, exists := podRes[container]
	if !exists {
		containerRes = corev1.ResourceRequirements{}
	}

	if limits != nil {
		containerRes.Limits = limits
	}
	if req != nil {
		containerRes.Requests = req
	}
	if claims != nil {
		containerRes.Claims = claims
	}

	podRes[container] = containerRes
	a.rules[podkey] = podRes
}

func (a *Adapter) getResourceRulesForContainer(podkey types.NamespacedName, container string) (corev1.ResourceList, corev1.ResourceList, []corev1.ResourceClaim) {

	a.m.RLock()
	defer a.m.RUnlock()

	podRes, exists := a.rules[podkey]
	if !exists {
		return nil, nil, nil
	}

	containerRes, exists := podRes[container]

	if exists {
		return containerRes.Requests, containerRes.Limits, containerRes.Claims
	}

	return nil, nil, nil
}
