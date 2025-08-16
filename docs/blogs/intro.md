# Improving NVidia Multi-Instance GPU Utilization in Kubernetes

The recent AI/LLM boom has spawned a plethora of services that requires GPUs. Those GPUs are really expensive, and for most services, taking up the entire GPU is a waste, a portion of the capability is sufficient to them.  NVidia as the leader of GPU vendors provided many solutions for concurrent use of a physical GPU. Among them, Multi-Instance GPU (MIG) provides production-ready capabilities like performance isolation, memory bandwidth QoS, error isolation, ... With MIG, administrators can group the Streaming Processors and Memory of a physical GPU as they like to provide capabilities fit to services. It is a good choice for applications need resiliency and QoS. 

Kubernetes is an open-source system for automating deployment, scaling and management of containerized applications. It provides access to special hardware such as GPUs through its device plugin framework. Once the plugin is in place, the cluster exposes custom schedulable resource for workloads to consume. 

The standard scenario to use MIG in Kubernetes is: cluster administrator partition the physical GPUs into tiny/small/medium/large instances; device plugin exposes those instances as different schedulable resource types; services request the resource type as needed; Kubernetes place the service on the node with available resource and start it. It looks just like the airlines offers first class, business, economy seats and travelers buys the tickets based on their needs.

Everything sounds great until the rubber hits the road.

## Unpredictable Demand in Dynamic Environments and Solution from Real Life

Demand for services is never even. In order to efficiently use resources, services are scaled on the current demand. With all the autoscalers in place, cluster administrator can never know what is the right number to prepare for each resource type. Such number does not exist in a dynamic environment, and the cloud providers ended up with having redundant resources for each type in cluster. Yes, there is option to re-partition physical GPU to shift resource among types, but the cost is high to drain out workloads on target physical device and reconfigure.

It looks the same as airlines again. They can never predict the number of customers for each ticket types, and it is way too expensive to change the cabin to the best layout based on sold tickets every time. The airlines encountered this problem much earlier and they came up with their solution for flight utilization: over sell and free upgrade. If there are over-sold economy customers while business or even first class seats are still available at departure, airlines let those economy customer take those seats for free so that both flight utilization and customer satisfaction are improved. 

A good reference for cloud providers!

## [MIG-Adapter](https://github.com/IBM/mig-adapter) in Details

To share the buffer resource across types, just like airlines share the available seats before departure, we introduce open source project [MIG Adapter](https://github.com/IBM/mig-adapter) to find the pending workloads in the cluster, to check the resource type they are waiting for and to upgrade it to a compatible and available type. Meanwhile, when certain resource type is freed up in the cluster, [MIG-Adapter](https://github.com/IBM/mig-adapter) also checks all the upgraded workloads and restore the applicable one back to original resource type. With MIG-Adapter, administrators reduce the total resource they need for buffering and improved the overall resource utilization. 

### Architecture and Strategy Adopted

MIG-Adapter consists of 3 main components: a Controller watching Pod resource, a MutatingAdmissionWebhook to patch Pod with new resource type and a in-memory RuleStore for Controller to pass the recommendation to MutatingAdmissionWebhook

Airlines start to check customers for upgrade right before the departure because by that time the status of the flight is stable. MIG-Adapter also seeks that kind of stableness to make the change. The timing picked is when the Pod enters "Pending" phase with a condition of insufficient resource. MIG-Adapter controller respond to that status and start to check compatible resource type to upgrade. If it fails to find anything, the Pod stays in the Pending status; but if it is lucky, controller will generate a rule for upgrade, then restart the pod to pick up that change. After the Pod is recreated, MIG-Adapter admission webhook finds the matching rule for it and patch the container resource requests/limits to the upgraded type. After a rules served its purpose, it is removed from rule store. Also the original resource demand is annotated to the Pod for future restore decisions.  The reason it goes through this restart/patch process is because Kubernetes does not allow in-place change to container resources. 

When there is a running Pod finished in the cluster or a new node added to the cluster, MIG-Adapter checks if it proves available MIG resource or not. If there is new available MIG resource, MIG-Adapter checks all upgraded workloads and find the applicable one to restart. That is the one wasted resource the most. For example, between a "mig-1g.5gb" upgraded to "mig-4g.20gb" and a "mig-2g.10gb" upgraded to "mig-3g.20gb", MIG-Adapter restores the first one when there is a "mig-2g.10gb" resource becomes available. After the new Pod is created, it will be patched with new resource type (like the example above) or use its original one to continue.

### Try MIG-Adapter Locally

There are two ways to run MIG-Adapter: deploy in the target cluster or start it locally. The common effort to run it is to generate required certificates for the mutating admission webhook. Different kubernetes providers have different ways of doing that. To make our life easier, we're describing the local approach in details here, it works as long as you and your target cluster can access each other.

#### Prepare Certificates, Manifests and Start MIG-Adapter

Signed certificates are required for webhook server. By default webhook server loads those certificates from `k8s-webhook-server/serving-certs/` under your `TMPDIR`. Let's create the folder and switch to it.

```shell
mkdir -p  "$TMPDIR"k8s-webhook-server/serving-certs/
cd "$TMPDIR"k8s-webhook-server/serving-certs/
```

##### Step 1. Generate CA key and certificates

```shell
openssl genrsa -out ca.key 2048
openssl req -new -x509 -days 365 -key ca.key \
  -subj "/C=AU/CN=simple-kubernetes-webhook"\
  -out ca.crt

```

##### Step 2. Generate Webhook server key and certificates

```shell
openssl req -newkey rsa:2048 -nodes -keyout server.key \
  -subj "/C=AU/CN=simple-kubernetes-webhook" \
  -out server.csr
cp server.key tls.key
```

##### Step 3. Sign Webhook server certificates with CA

```shell
openssl x509 -req \
  -extfile <(printf "subjectAltName=IP:192.168.2.14") \
  -days 365 \
  -in server.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out tls.crt
```

##### Step 4. Update Manifests with CA certificates

```shell
cat ca.crt | base64 
```

After all certificates are generated, let's go back to project folder

```shell
cd -
```

Update the `webhooks[0].clientConfig.CABundle` and `webhooks[0].clientConfig.url` [manifest file](../../config/webhook/manifests.yaml).

Please be advised that `TMPDIR` are cleaned up by your local os from time to time, you may need to redo the steps above.

##### Step 5. Apply webhook manifests and start MIG-Adapter

Make sure you're in your project folder and then 

```shell
kubectl apply -f config/webhook/manifests.yaml
make run
```

#### Upgrade Workloads

Assuming there is 1 x A100 NVidia GPU in the target cluster cluster and it is configured with `all-balanced` profile. That means 2 x "mig-1g.5gb", 1 x "mig-2g.10gb", 1 x "mig-3g.20gb" resources in the cluster. 

Lets apply start with 2 workloads asking for 1 x "mig-1g.5gb" each. 

```shell
kubectl apply -f test/workloads/simple-1g-1.yaml
kubectl apply -f test/workloads/simple-1g-2.yaml

```

They'are all running.

```shell
kubectl get pods -o json | jq 'items[] | { name: .metadata.name, status: .status.phase, resources: .spec.containers[].resources}'
{
    "name": "workload-1g-1-776f69849-x2h5p",
    "status" : "Running",
    "resources": {
        "limits": {
            "nvidia.com/mig-1g.5gb": "1"
        },
        "requests": {
            "nvidia.com/mig-1g.5gb": "1"
        },
    }
}
{
    "name": "workload-1g-2-7787ff5678-66vsx",
    "status" : "Running",
    "resources": {
        "limits": {
            "nvidia.com/mig-1g.5gb": "1"
        },
        "requests": {
            "nvidia.com/mig-1g.5gb": "1"
        },
    }
}
```

Then we scale out workload-1g-2.

```shell
kubectl scale deploy workload-1g-2 --replicas=2
```

The new Pod suppose to be in `Pending` status, but with the help from MIG-Adapter, the resource demand is upgraded to "mig-2g.10gb", and it is able to run.

```shell
kubectl get pods -o json | jq 'items[] | { name: .metadata.name, status: .status.phase, resources: .spec.containers[].resources}'
{
    "name": "workload-1g-1-776f69849-x2h5p",
    "status" : "Running",
    "resources": {
        "limits": {
            "nvidia.com/mig-1g.5gb": "1"
        },
        "requests": {
            "nvidia.com/mig-1g.5gb": "1"
        },
    }
}
{
    "name": "workload-1g-2-7787ff5678-66vsx",
    "status" : "Running",
    "resources": {
        "limits": {
            "nvidia.com/mig-1g.5gb": "1"
        },
        "requests": {
            "nvidia.com/mig-1g.5gb": "1"
        },
    }
}
{
    "name": "workload-1g-2-7787ff5678-9kwgs",
    "status" : "Running",
    "resources": {
        "limits": {
            "nvidia.com/mig-2g.10gb": "1"
        },
        "requests": {
            "nvidia.com/mig-2g.10gb": "1"
        },
    }
}
```

#### Restore Workloads

Let's delete the workload-1g-1 now and check the Pods

```shell
kubectl delete -f -f test/workloads/simple-1g-1.yaml
```

It takes some time to complete the Pod deletion and to free the "mig-1g.5gb" allocated to it. Eventually, the upgraded Pod goes back to its original resource demand.

```shell
kubectl get pods -o json | jq 'items[] | { name: .metadata.name, status: .status.phase, resources: .spec.containers[].resources}'
{
    "name": "workload-1g-2-7787ff5678-66vsx",
    "status" : "Running",
    "resources": {
        "limits": {
            "nvidia.com/mig-1g.5gb": "1"
        },
        "requests": {
            "nvidia.com/mig-1g.5gb": "1"
        },
    }
}
{
    "name": "workload-1g-2-7787ff5678-hxnq9",
    "status" : "Running",
    "resources": {
        "limits": {
            "nvidia.com/mig-1g.5gb": "1"
        },
        "requests": {
            "nvidia.com/mig-1g.5gb": "1"
        },
    }
}

```

#### Clean up MIG-Adapter

After stopping the MIG-Adapter process, don't forget to cleanup the webhook manifests in your target cluster.

```shell
kubectl delete -f config/webhook/manifests.yaml
```

### Other considerations

Current MIG-Adapter implements the basic but foundational feature. There are certainly lots of improvements can be added. Feel free to create an issue in MIG-Adapter [repository](https://github.com/IBM/mig-adapter) to add yours

* Annotate workloads don't want to be upgraded or don't want to be restored
* Repartition MIG on GPU to satisfy workloads
* Timeout stale patching rules
* Defragment allocated MIG instances
* Consider other constraints when recommending new resource types
* Throttle upgraded workloads to use resource no more than its original demand

## Next Steps

Now you understand how to use MIG-Adapter to improve the utilization of your NVidia MIG in your Kubernetes cluster, go ahead and try it yourself.