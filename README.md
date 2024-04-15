# mig-adapter

a tool to automatically adjust kubernetes pod resource requests/limits based on available nvidia MIG instances

## Overview

Kubernetes is an open-source system for automating deployment, scaling and management of containerized applications. Kubernetes provides access to special hardware such as GPUs, NICs through the device plugin framework. 

Multi-Instance GPU (MIG) is a technology provided by NVidia to expand the performance and value of some of its GPU products. It can partition GPU to multiple instances, each fully isolated with its own high-bandwidth memory, cache and compute cores. This gives administrators the ability to support every workload, from the smallest to the largest, with guaranteed quality of service (QoS) and extending the reach of accelerated computing resources to every user.

The NVIDIA GPU Operator uses the operator framework within Kubernetes to automate the management of all NVIDIA software components needed to provision GPU including MIG. Once the operator is installed, the cluster exposes custom schedulable resources such as `nvidia.com/gpu`, `nvidia.com/mig-1g.5gb`, ... Workload can consume these GPUs/MIGs from containers by requesting custom resource the same way to request `cpu` or `memory`, but without dynamic MIG device allocation there is no way to tell kubernetes scheduler to pick one from a set of resources types. i.e. In a cluster with mixed MIG strategy, there is no way to tell the kuberentes scheduler to pick either 1 `nvidia.com/mig-1g.10gb` or 1 `nvidia.com/mig-2g.10gb`.  This introduces problem to system utilization: in a dynamic environment, there no way to predict the demand for each of the MIG type. 

MIG Adapter is trying to solve this problem with static MIG device allocation by adjusting the workload demand from an insufficient resource type to a compatible and available resource type. The compatibility is decided by the compute 

## Architecture

MIG Adapter consists of 3 components: 

1. A Controller to 
    * Seek Pending Pods with Insufficient MIG resources, generate rule to patch them to new resource type
    * After MIG resources is freed, restart applicable Pods to change back to original resource type
2. A RuleStore to keep the rules to patch Pods
3. A Mutating Admission Webhook to patch Pods based on rules generated by Controller

## Quick Start

First of all, create the organization directory and clone this project

```shell
mkdir IBM
cd IBM
git clone https://github.ibm.com/IBM/mig-adapter.git
cd mig-adapter
```

There are two ways to run MIG Adapter

### Run Locally (Recommended)

Technically, the webhooks can be run locally, but for it to work you need to generate certificates for the webhook server and store them at /tmp/k8s-webhook-server/serving-certs/tls.{crt,key}. For more details about running webhook locally, refer [here](https://book.kubebuilder.io/cronjob-tutorial/running.html#running-webhooks-locally).

Some shell commands to assist certificates generation are kept [here](hack/gencert.sh)

The [MutatingWebhookConfiguration](config/webhook/manifests.yaml) also needs to be updated with the generated certificates

### Run as a Deployment inside the cluster

Running MIG Adapter as a Deployment inside the cluster is the same to deploying an Operator. For instructions on deploying MIG Adapter into a cluster, refer to the Operator SDK [tutorial](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/#2-run-as-a-deployment-inside-the-cluster).

If the target cluster is a OpenShift cluster, refer to its [doc](https://docs.openshift.com/container-platform/4.15/security/certificates/service-serving-certificate.html) for injecting certificates.