# VeriSilicon Solios-X device plugin for Kubernetes

# Table of Contents

* [Introduction](#introduction)
* [Verify node kubelet config](#Verify-node-kubelet-config)
* [Install Solios-X driver](#Install-Solios-X-driver)
* [Deploy plugin DaemonSet](#Deploy-plugin-DaemonSet)
* [Label your server](#Label-your-server)
* [Testing the plugin by Deployment](#Testing-the-plugin-by-Deployment)
* [Or Testing the plugin by Pod](#Or-Testing-the-plugin-by-Pod)

# Introduction

The Solios-X device plugin for Kubernetes supports acceleration using VeriSilicon Solios-X solution.
The following sections detail how to obtain, build, deploy and test the Solios-X device plugin.
Examples are provided showing how to deploy the plugin, and test the plugin by POD and Deployment.

## Verify node kubelet config

Every node that will be running the Solios-X plugin must have the
[kubelet device-plugins](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/)
configured. For each node, check that the kubelet device plugin socket exists:

```bash
$ ls /var/lib/kubelet/device-plugins/kubelet.sock
/var/lib/kubelet/device-plugins/kubelet.sock
```

## Install Solios-X driver
Please Make sure to install Solios-X driver on your server, you can follow https://github.com/VeriSilicon/vpe/blob/vs_develop/readme.md to make and install Solios-X driver.
After the driver is installed, please double check whether device nodes /dev/transcoderxx are available. If 
/dev/transcoderxx are available then it means driver was installed successfully.

If your driver was already installed, please skip this step.

## Deploy plugin DaemonSet

```bash
$ kubectl apply -f https://raw.githubusercontent.com/VeriSilicon/solios-x-device-plugin/master/deployments/solios-x-device-plugin.yaml
```

## Label your server

Please change [NODE NAME] to your node name. you can use "kubectl get nodes" to get you node name.

```bash
$ kubectl label nodes [NODE NAME] solios-device=enable
```
After this step, you will able to find the verisilicon.com/solios resources already been reported by plugin if everything went smooth:
```bash
$ kubectl describe nodes [NODE NAME]
...
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
$ kubectl apply -f https://raw.githubusercontent.com/VeriSilicon/solios-x-device-plugin/master/deployments/solios-x-test-deployment-csd.yaml
```
In this sample YAML file, 5 Solios-X cards will be used hence 5 pods will be created. If you don't have 10 cards installed on your server, please change [replicas] value.
The sample pod will select one /dev/transcoder device and do H264->H264 transcoding job repeatedly.
```bash
apiVersion: apps/v1 # for versions before 1.9.0 use apps/v1beta2
kind: Deployment
metadata:
  name: solios-test-deployment
spec:
  replicas: 5

```

check pod status:
```bash
$kubectl get pods
NAME                                      READY   STATUS    RESTARTS   AGE
solios-test-deployment-574d464658-d79q9   1/1     Running   0          111s
solios-test-deployment-574d464658-n42w9   1/1     Running   0          111s
solios-test-deployment-574d464658-x2fsl   1/1     Running   0          111s
solios-test-deployment-574d464658-znv8z   1/1     Running   0          111s
solios-test-deployment-574d464658-zpj9b   1/1     Running   0          111s
```

check one pod log:
```bash
$kubectl logs solios-test-deployment-574d464658-d79q9
found device  /dev/transcoder2
POD  Doing transcoding...device=/dev/transcoder2, count = 1, output_file = output0_.h264
ffmpeg version N-98227-g45bddede35 Copyright (c) 2000-2020 the FFmpeg developers
  built with gcc 4.8.5 (GCC) 20150623 (Red Hat 4.8.5-39)
  configuration: --enable-vpe --extra-ldflags=-L/lib/vpe --extra-libs=-lvpi --disable-sdl2 --disable-libxcb --disable-libxcb-shm --disable-libxcb-xfixes --disable-libxcb-shape --disable-xlib --disable-libmfx --disable-vaapi
  libavutil      56. 55.100 / 56. 55.100
  libavcodec     58. 92.100 / 58. 92.100
  libavformat    58. 46.101 / 58. 46.101
  libavdevice    58. 11.100 / 58. 11.100
  libavfilter     7. 86.100 /  7. 86.100
  libswscale      5.  8.100 /  5.  8.100
  libswresample   3.  8.100 /  3.  8.100
[h264 @ 0x3f9ce40] Stream #0: not enough frames to estimate rate; consider increasing probesize
Input #0, h264, from 'test1080p.h264':
  Duration: N/A, bitrate: N/A
    Stream #0:0: Video: h264 (Main), yuv420p(tv, bt709, progressive), 1920x1080, 29.97 fps, 29.97 tbr, 1200k tbn, 59.94 tbc
Stream mapping:
  Stream #0:0 (h264_vpe) -> spliter_vpe
  spliter_vpe -> Stream #0:0 (h264enc_vpe)
Press [q] to stop, [?] for help
Output #0, h264, to 'output0_.h264':
  Metadata:
    encoder         : Lavf58.46.101
    Stream #0:0: Video: h264 (h264enc_vpe), vpe, 1920x1080, q=2-31, 10000 kb/s, 29.97 fps, 29.97 tbn, 29.97 tbc
    Metadata:
      encoder         : Lavc58.92.100 h264enc_vpe
frame=  300 fps=175 q=-0.0 Lsize=   12020kB time=00:00:09.74 bitrate=10106.9kbits/s speed=5.69x
```

Delete the Deployment:
```bash
$kubectl delete -f https://raw.githubusercontent.com/VeriSilicon/solios-x-device-plugin/master/deployments/solios-x-test-deployment-csd.yaml
deployment.apps "solios-test-deployment" deleted
```

## Or Testing the plugin by Pod

```bash
$ kubectl apply -f https://raw.githubusercontent.com/VeriSilicon/solios-x-device-plugin/master/deployments/solios-x-test-pod-csd.yaml
```
In this sample YAML file, 1 Solios-X cards will be used hence one 1 pods will be created.
The sample pod will select one /dev/transcoder device and do H264->H264 transcoding job repeatedly.

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
Check the pod log:
```bash
$kubectl logs solios-test-pod
found device  /dev/transcoder7
POD 10.244.0.121 Doing transcoding...count = 1, output_file = output0_10.244.0.121.h264
ffmpeg version N-98227-g45bddede35 Copyright (c) 2000-2020 the FFmpeg developers
  built with gcc 4.8.5 (GCC) 20150623 (Red Hat 4.8.5-39)
  configuration: --enable-vpe --extra-ldflags=-L/lib/vpe --extra-libs=-lvpi --disable-sdl2 --disable-libxcb --disable-libxcb-shm --disable-libxcb-xfixes --disable-libxcb-shape --disable-xlib --disable-libmfx --disable-vaapi
  libavutil      56. 55.100 / 56. 55.100
  libavcodec     58. 92.100 / 58. 92.100
  libavformat    58. 46.101 / 58. 46.101
  libavdevice    58. 11.100 / 58. 11.100
  libavfilter     7. 86.100 /  7. 86.100
  libswscale      5.  8.100 /  5.  8.100
  libswresample   3.  8.100 /  3.  8.100
[h264 @ 0x257ae40] Stream #0: not enough frames to estimate rate; consider increasing probesize
Input #0, h264, from 'test1080p.h264':
  Duration: N/A, bitrate: N/A
    Stream #0:0: Video: h264 (Main), yuv420p(tv, bt709, progressive), 1920x1080, 29.97 fps, 29.97 tbr, 1200k tbn, 59.94 tbc
Stream mapping:
  Stream #0:0 (h264_vpe) -> spliter_vpe
  spliter_vpe -> Stream #0:0 (h264enc_vpe)
Press [q] to stop, [?] for help
Output #0, h264, to 'output0_10.244.0.121.h264':
  Metadata:
    encoder         : Lavf58.46.101
    Stream #0:0: Video: h264 (h264enc_vpe), vpe, 1920x1080, q=2-31, 10000 kb/s, 29.97 fps, 29.97 tbn, 29.97 tbc
    Metadata:
      encoder         : Lavc58.92.100 h264enc_vpe
frame=  300 fps=181 q=-0.0 Lsize=   12020kB time=00:00:09.74 bitrate=10106.9kbits/s speed=5.88x
video:12020kB audio:0kB subtitle:0kB other streams:0kB global headers:0kB muxing overhead: 0.000000%
(null)[-1][9][9][L6][Trans_mem]TransCheckMemLeakInit(91): init_trans_memory test
(null)[07][9][9][L6][TRANS_FD]TranscodeOpenFD(104): fd opend 3
log_filename vpi_20200903_044946.log
log info send to syslog
system[07][9][9][L4][DECDWL0]dechw_c0 count: 161, total: 1119707, 6954 pertime
system[07][9][9][L4][DECDWL0]dechw_c1 count: 107, total: 739606, 6912 pertime
system[07][9][9][L4][DECDWL0]dechw_c2 count: 27, total: 186086, 6892 pertime
system[07][9][9][L4][DECDWL0]dechw_c3 count: 5, total: 35422, 7084 pertime
system[07][9][9][L4][DECDWL0]dechw count: 77, total: 1428861, 18556 pertime
system[07][9][9][L4][H264ENC]vcehwp1 count: 0, total: 0, 0 pertime
system[07][9][9][L4][H264ENC]vcehw count: 300, total: 1095228, 3650 pertime
system[07][9][9][L4][H264ENC]CU_ANAL count: 0, total: 0, 0 pertime
system[07][9][9][L4][H264ENC]vcehw_total count: 300, total: 1096033, 3653 pertime
system[07][9][9][L4][H264ENC]vce_total count: 300, total: 1247043, 4156 pertime
Round 1 finished, now start next...

```
Delete the Pod:
```bash
$kubectl delete -f https://raw.githubusercontent.com/VeriSilicon/solios-x-device-plugin/master/deployments/solios-x-test-pod-csd.yaml
deployment.apps "solios-test-pod-csd" deleted
```
