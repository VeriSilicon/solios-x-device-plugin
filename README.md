# VeriSilicon Solios-X device plugin for Kubernetes

# Table of Contents

* [Introduction](#introduction)
* [Installation](#installation)
    * [Getting the source code](#getting-the-source-code)
    * [Verify node kubelet config](#verify-node-kubelet-config)
    * [Deploying as a DaemonSet](#deploying-as-a-daemonset)
        * [Build the plugin image](#build-the-plugin-image)
        * [Deploy plugin DaemonSet](#deploy-plugin-daemonset)
    * [Deploy by hand](#deploy-by-hand)
        * [Build the plugin](#build-the-plugin)
        * [Run the plugin as administrator](#run-the-plugin-as-administrator)
    * [Verify plugin registration](#verify-plugin-registration)
    * [Testing the plugin](#testing-the-plugin)

# Introduction

The Solios-X device plugin for Kubernetes supports acceleration using VeriSilicon Solios-X solution.

# Installation

The following sections detail how to obtain, build, deploy and test the Solios-X device plugin.

Examples are provided showing how to deploy the plugin either using a DaemonSet or by hand on a per-node basis.

## 1. Verify node kubelet config

Every node that will be running the Solios-X plugin must have the
[kubelet device-plugins](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/)
configured. For each node, check that the kubelet device plugin socket exists:

```bash
$ ls /var/lib/kubelet/device-plugins/kubelet.sock
/var/lib/kubelet/device-plugins/kubelet.sock
```

## 2. Get Solios-X device plugin source code:

```bash
$ git clone git@github.com:VeriSilicon/solios-x-device-plugin.git
Cloning into 'solios-x-device-plugin'...
remote: Enumerating objects: 48, done.
remote: Counting objects: 100% (48/48), done.
remote: Compressing objects: 100% (27/27), done.
Receiving objects: 100% (48/48), 30.90 KiB | 179.00 KiB/s, done.
remote: Total 48 (delta 10), reused 47 (delta 9), pack-reused 0
Resolving deltas: 100% (10/10), done.
```

## 3. Deploy plugin DaemonSet

```bash
$ kubectl apply -f solios-x-device-plugin.yaml
```

## 4. Label your server:

```bash
$ kubectl label nodes [NODE NAME] solios-device=enable
```
After step 5, you will able to find the verisilicon.com/solios resources already been reported by plugin if everything went smooth:
```bash
$ kubectl describe nodes vsi
Name:               vsi
Roles:              <none>
Labels:             kubernetes.io/arch=amd64
                    kubernetes.io/hostname=vsi
                    kubernetes.io/os=linux
                    solios-device=enable
...
Capacity:
  cpu:                     36
  ephemeral-storage:       51175Mi
  github.com/fuse:         5k
  hostdev.k8s.io/dev_mem:  1
  hugepages-1Gi:           0
  hugepages-2Mi:           10Gi
  memory:                  65617948Ki
  pods:                    110
  verisilicon.com/solios:  10

```

## 5. Testing the plugin by Deployment

```bash
$ kubectl apply -f deployments/solios-x-test-deployment-csd.yaml
```
In this sample YAML file, 10 Solios-X cards will be used hence 10 pods will be created. If you don't have 10 cards installed on your server, please change [replicas] value.
```bash
apiVersion: apps/v1 # for versions before 1.9.0 use apps/v1beta2
kind: Deployment
metadata:
  name: solios-test-deployment
spec:
  replicas: 10

```

## 6. Testing the plugin by Pod

```bash
$ kubectl apply -f deployments/solios-x-test-pod-csd.yaml
```
In this sample YAML file, 1 Solios-X cards will be used hence one 1 pods will be created.
If you's like to select multiple cards in one pod, you can change verisilicon.com/solios: values from 1 to others.

In below example, 2 Solios-x cards will be selected in one pods:

```bash
    resources:
      requests:
        verisilicon.com/solios: 2
      limits:
        memory: "500Mi"
        hugepages-2Mi: 1024Mi
        cpu: 4
        verisilicon.com/solios: 2
```
