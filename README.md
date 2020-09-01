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

## 1. Getting the source code

> **Note:** It is presumed you have a valid and configured [golang](https://golang.org/) environment
> that meets the minimum required version.

```bash
$ mkdir -p $(go env GOPATH)/src/github.com/intel
$ git clone https://github.com/intel/intel-device-plugins-for-kubernetes $(go env GOPATH)/src/github.com/intel/intel-device-plugins-for-kubernetes
```

## 2. Verify node kubelet config

Every node that will be running the gpu plugin must have the
[kubelet device-plugins](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/)
configured. For each node, check that the kubelet device plugin socket exists:

```bash
$ ls /var/lib/kubelet/device-plugins/kubelet.sock
/var/lib/kubelet/device-plugins/kubelet.sock
```

## 3. Get Solios-X device plugin source code:

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

## 4. Deploy plugin DaemonSet

```bash
$ kubectl apply -f solios-x-device-plugin.yaml
```

## 5. Label your server:

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

## Testing the plugin by Deployment

```bash
$ kubectl apply -f solios-x-test-deployment.yaml
```

> **Note**: It is also possible to run the GPU device plugin using a non-root user. To do this,
the nodes' DAC rules must be configured to device plugin socket creation and kubelet registration.
Furthermore, the deployments `securityContext` must be configured with appropriate `runAsUser/runAsGroup`.

## Testing the plugin by Pod

```bash
$ kubectl apply -f solios-x-test-pod.yaml
```

> **Note**: It is also possible to run the GPU device plugin using a non-root user. To do this,
the nodes' DAC rules must be configured to device plugin socket creation and kubelet registration.
Furthermore, the deployments `securityContext` must be configured with appropriate `runAsUser/runAsGroup`.
