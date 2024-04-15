package adapter

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	RESOURCE_MIG_PREFIX                = "nvidia.com/mig-"
	PODMESSAGE_INSUFFICIENT_PREFIX     = "Insufficient "
	PODMESSAGE_INSUFFICIENT_MIG_PREFIX = PODMESSAGE_INSUFFICIENT_PREFIX + RESOURCE_MIG_PREFIX

	LABELKEY_MIG_CONFIG = "nvidia.com/mig.config"
)

var amlog = logf.Log.WithName("adapter mig")

const (
	MIG_FORMAT = RESOURCE_MIG_PREFIX + "%dg.%dgb"
)

type migIdentifier struct {
	Compute int
	Memory  int
}

func (d migIdentifier) Equal(target *migIdentifier) bool {
	if d.Compute == target.Compute && d.Memory == target.Memory {
		return true
	}

	return false
}

func (d migIdentifier) Less(target *migIdentifier) bool {
	if d.Compute < target.Compute {
		return true
	}

	if d.Compute == target.Compute && d.Memory < target.Memory {
		return true
	}

	return false
}

func (m *migIdentifier) Parse(str string) error {
	n, err := fmt.Sscanf(str, MIG_FORMAT, &m.Compute, &m.Memory)

	if n != 2 {
		return fmt.Errorf("expect 2 but only scanned %d for mig", n)
	}

	return err
}

func (m *migIdentifier) String() string {
	return fmt.Sprintf(MIG_FORMAT, m.Compute, m.Memory)
}

type OrderedmigIdentifierList []migIdentifier

func (o OrderedmigIdentifierList) Len() int {
	return len(o)
}

// Less compare compute and then memory
func (o OrderedmigIdentifierList) Less(i, j int) bool {

	return o[i].Less(&o[j])
}

func (o OrderedmigIdentifierList) Swap(i, j int) {

	var tmp migIdentifier

	tmp.Compute = o[i].Compute
	tmp.Memory = o[i].Memory

	o[i].Compute = o[j].Compute
	o[i].Memory = o[j].Memory

	o[j].Compute = tmp.Compute
	o[j].Memory = tmp.Memory

}

type availableMIGsOnNode struct {
	NodeLabels map[string]string
	MIGs       map[migIdentifier]resource.Quantity
}

type availableMIGMap map[string]availableMIGsOnNode

type podDescriptor struct {
	md  *migIdentifier
	Pod *corev1.Pod
}

func (m *podDescriptor) ParsePod(pod *corev1.Pod) error {
	for _, c := range pod.Spec.Containers {
		for k := range c.Resources.Limits {
			if strings.Contains(k.String(), RESOURCE_MIG_PREFIX) {
				m.Pod = pod
				if m.md == nil {
					m.md = &migIdentifier{}
				}
				m.md.Parse(k.String())

				// assuming there is only 1 container has mig req and that container only has 1 mig req
				return nil
			}
		}
	}

	return errors.New("No MIG Resource In Pod " + pod.Namespace + "/" + pod.Name)
}

type podDescriptorList []podDescriptor

func (l podDescriptorList) Len() int {
	return len(l)
}

// Less compare compute and then memory
func (l podDescriptorList) Less(i, j int) bool {

	return l[i].md.Less(l[j].md)
}

func (l podDescriptorList) Swap(i, j int) {
	var tmp podDescriptor

	tmp.md = l[i].md
	tmp.Pod = l[i].Pod

	l[i].md = l[j].md
	l[i].Pod = l[j].Pod

	l[j].md = tmp.md
	l[j].Pod = tmp.Pod
}

// replace the mig resource in the list, return false if it is the same
// keep 1 and only 1 mig resource in the list
func (a *Adapter) updateMIGInResourceList(list corev1.ResourceList, mig *migIdentifier, q resource.Quantity) bool {

	n := mig.String()
	for k, v := range list {
		// remove the
		if strings.Contains(k.String(), RESOURCE_MIG_PREFIX) && k.String() != n {
			delete(list, k)
		} else if k.String() == n && v.Equal(q) {
			return false
		}
	}

	list[corev1.ResourceName(n)] = q

	return true
}

/*
	   -1 : <
		0 : =
		1 : >
*/
func (a *Adapter) compareMIGResources(src, dst corev1.ResourceList) int {
	srcmd, _ := a.currentMIGResource(src)
	dstmd, _ := a.currentMIGResource(dst)

	if srcmd.Less(dstmd) {
		return -1
	}

	if srcmd.Equal(dstmd) {
		return 0
	}

	return 1
}

func (a *Adapter) currentMIGResource(list corev1.ResourceList) (*migIdentifier, *resource.Quantity) {

	// assume only 1 mig entry in resource list
	for k, v := range list {
		if strings.Contains(k.String(), RESOURCE_MIG_PREFIX) {
			md := &migIdentifier{}
			md.Parse(k.String())
			return md, &v
		}
	}

	return nil, nil
}

func (a *Adapter) findAvailableMIGResource(current *migIdentifier, quantity resource.Quantity, selector map[string]string, available availableMIGMap, order OrderedmigIdentifierList) *migIdentifier {

	located := false
	for _, n := range order {
		if !located {
			if current.Less(&n) || current.Equal(&n) {
				located = true
			}
		}
		if located {
			for node, migsOnNode := range available {

				matched := true
				for k, v := range selector {
					if migsOnNode.NodeLabels[k] != v {
						matched = false
						break
					}
				}
				if !matched {
					continue
				}

				q := migsOnNode.MIGs[n]
				if q.Cmp(quantity) != -1 {
					// remove the resource from available
					q.Sub(quantity)
					migsOnNode.MIGs[n] = q
					available[node] = migsOnNode

					return &n
				}
			}
		}
	}

	return nil
}

func (a *Adapter) getAvailableMIGsAndOrder(nodes []corev1.Node, pods []corev1.Pod) (availableMIGMap, OrderedmigIdentifierList) {
	available := a.detectAllAvailableMIGs(nodes, pods)
	if len(available) == 0 {
		return nil, nil
	}
	order := a.buildOrderedMIGList(available)

	return available, order
}

// static list for now
// the process of building the list
// get all keys from availableMIGMap
// sort all keys into the ordered list
func (a *Adapter) buildOrderedMIGList(available availableMIGMap) OrderedmigIdentifierList {

	if available == nil {
		return nil
	}

	order := OrderedmigIdentifierList{}
	for _, node := range available {
		for mig := range node.MIGs {
			md := &migIdentifier{}
			md.Parse(mig.String())
			order = append(order, *md)
		}
	}

	sort.Sort(order)

	return order
}

// static map for now
// the process of discovering and loading the map
// 1. get clusterpolicies.nvidia.com  (cluster resource)
// 2. from cluster policy get the configmap name from spec.migManager.config.name
// 3. get the configmap data
func (a *Adapter) buildMIGProfileMap(cfg *corev1.ConfigMap) map[corev1.ResourceName]string {

	if cfg == nil {
		return map[corev1.ResourceName]string{
			corev1.ResourceName(RESOURCE_MIG_PREFIX + "1g.5gb"):  "all-1g.5gb",
			corev1.ResourceName(RESOURCE_MIG_PREFIX + "2g.10gb"): "all-2g.10gb",
			corev1.ResourceName(RESOURCE_MIG_PREFIX + "3g.20gb"): "all-3g.20gb",
			corev1.ResourceName(RESOURCE_MIG_PREFIX + "4g.20gb"): "all-4g.20gb",
		}
	}

	return nil
}

func (a *Adapter) getAllocableMIGsOnNode(node *corev1.Node) availableMIGsOnNode {
	migsOnNode := availableMIGsOnNode{
		NodeLabels: node.Labels,
		MIGs:       make(map[migIdentifier]resource.Quantity),
	}

	for k, v := range node.Status.Allocatable {
		if strings.Contains(k.String(), RESOURCE_MIG_PREFIX) {
			md := &migIdentifier{}
			md.Parse(k.String())
			migsOnNode.MIGs[*md] = v.DeepCopy()
		}
	}

	return migsOnNode
}

func (a *Adapter) detectAllAvailableMIGs(nodes []corev1.Node, pods []corev1.Pod) availableMIGMap {
	available := make(map[string]availableMIGsOnNode)

	// Get all allocable
	for _, node := range nodes {
		available[node.Name] = a.getAllocableMIGsOnNode(&node)
	}

	for _, pod := range pods {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		for _, c := range pod.Spec.Containers {
			for k, v := range c.Resources.Limits {
				if strings.Contains(k.String(), RESOURCE_MIG_PREFIX) {
					md := &migIdentifier{}
					md.Parse(k.String())
					q := available[pod.Spec.NodeName].MIGs[*md]
					q.Sub(v)
					available[pod.Spec.NodeName].MIGs[*md] = q
				}
			}
		}
	}

	amlog.Info("detect available migs", "migs", len(available))

	return available
}

func (a *Adapter) checkAndSizeUpMIGForContainerResource(req, limits corev1.ResourceList, selector map[string]string, available availableMIGMap, order OrderedmigIdentifierList) bool {

	current, quantity := a.currentMIGResource(req)
	if current == nil || quantity == nil {
		current, quantity = a.currentMIGResource(limits)
	}
	if current == nil {
		amlog.Info("failed to find current mig", "req", req, "limits", limits)
		return false
	}

	newmig := a.findAvailableMIGResource(current, *quantity, selector, available, order)
	if newmig == nil {
		amlog.Info("no available mig to size up")
		return false
	}

	updated := a.updateMIGInResourceList(req, newmig, *quantity)
	if !updated {
		return false
	}

	a.updateMIGInResourceList(limits, newmig, *quantity)
	return true
}

func (a *Adapter) findAvailableNodeWithFreeGPU(selector map[string]string, nodes []corev1.Node, available availableMIGMap) *corev1.Node {

	selectedNodes := []*corev1.Node{}

	for _, n := range nodes {
		selected := true
		for k, v := range selector {
			if n.Labels[k] != v {
				selected = false
			}
		}
		if selected {
			selectedNodes = append(selectedNodes, n.DeepCopy())
		}
	}

	// for now: assume only 1 GPU on 1 Node -> free means no pod usage
	for _, n := range selectedNodes {
		if candidate, ok := available[n.Name]; ok {
			if reflect.DeepEqual(a.getAllocableMIGsOnNode(n).MIGs, candidate.MIGs) {
				return n.DeepCopy()
			}
		}
	}

	return nil
}

// desc sort pods by MIG
func (a *Adapter) filterAndSortPodsDescendingByMIG(pods []corev1.Pod) []*corev1.Pod {

	l := podDescriptorList{}
	for i := range pods {
		pd := &podDescriptor{}
		err := pd.ParsePod(&pods[i])
		if err == nil {
			l = append(l, *pd)
		}
	}

	sort.Sort(l)
	sorted := []*corev1.Pod{}

	for _, pd := range l {
		prev := sorted
		sorted = []*corev1.Pod{pd.Pod}
		sorted = append(sorted, prev...)
	}

	return sorted
}
