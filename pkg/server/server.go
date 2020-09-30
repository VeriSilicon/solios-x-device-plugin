/*
 * Copyright (c) 2020, VeriSilicon Holdings Co., Ltd. All rights reserved
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package server

//#include "srm.h"
import "C"

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	RESOURCE_NAME_SOLIOS string = "verisilicon.com/solios"
	RESOURCE_NAME_480P   string = "verisilicon.com/solios_480p"
	RESOURCE_NAME_720P   string = "verisilicon.com/solios_720p"
	RESOURCE_NAME_1080P  string = "verisilicon.com/solios_1080p"
	RESOURCE_NAME_2160P  string = "verisilicon.com/solios_2160p"
	LOCATION             string = "/dev"
	SOLIOS_SOCKET        string = "solios.sock"
	SOLIOS_480P_SOCKET   string = "solios_480p.sock"
	SOLIOS_720P_SOCKET   string = "solios_720p.sock"
	SOLIOS_1080P_SOCKET  string = "solios_1080p.sock"
	SOLIOS_2160P_SOCKET  string = "solios_2160p.sock"
	SOLIOS_DEVICE_PREFIX string = "transcoder"
	KUBELET_SOCKET       string = "kubelet.sock"
	DEVICE_PLUG_PATH     string = "/var/lib/kubelet/device-plugins/"
)

type SoliosServer struct {
	srv             *grpc.Server
	devices         map[string]*pluginapi.Device
	notify          chan bool
	ctx             context.Context
	cancel          context.CancelFunc
	socket_name     string
	allocation_unit int //0 - solios, 1 - 480p, 2- 720p , 3- 1080p, 4 - 2160p
	power_mode      int //0 - Fix mode, 1 - Power Saving mode
	res_name        string
}

func NewSoliosServer() *SoliosServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &SoliosServer{
		devices:         make(map[string]*pluginapi.Device),
		srv:             grpc.NewServer(grpc.EmptyServerOption{}),
		notify:          make(chan bool),
		ctx:             ctx,
		cancel:          cancel,
		socket_name:     SOLIOS_SOCKET,
		allocation_unit: 0, //defailt = solios
		power_mode:      0, //defailt = BALANCE
	}
}

func (s *SoliosServer) Run(allocation_unit string, power_mode string) error {

	//get input parameters
	if strings.Compare(allocation_unit, "solios") == 0 {
		s.allocation_unit = 0
	} else if strings.Compare(allocation_unit, "480p") == 0 {
		s.allocation_unit = 1
	} else if strings.Compare(allocation_unit, "720p") == 0 {
		s.allocation_unit = 2
	} else if strings.Compare(allocation_unit, "1080p") == 0 {
		s.allocation_unit = 3
	} else if strings.Compare(allocation_unit, "2160p") == 0 {
		s.allocation_unit = 4
	} else {
		log.Infoln("Invalid allocation_unit string, set allocation_unit to solios")
	}

	if strings.Compare(power_mode, "power_saving") == 0 {
		s.power_mode = 1
	} else if strings.Compare(power_mode, "balance") == 0 {
		s.power_mode = 0
	} else {
		log.Infoln("Invalid power_mode string, set power_mode to power_saving")
	}

	log.Printf("Run, allocation_unit=%d, power_mode=%d", s.allocation_unit, s.power_mode)

	if s.allocation_unit == 0 {
		s.res_name = RESOURCE_NAME_SOLIOS
		s.socket_name = SOLIOS_SOCKET
	} else if s.allocation_unit == 1 {
		s.res_name = RESOURCE_NAME_480P
		s.socket_name = SOLIOS_480P_SOCKET
	} else if s.allocation_unit == 2 {
		s.res_name = RESOURCE_NAME_720P
		s.socket_name = SOLIOS_720P_SOCKET
	} else if s.allocation_unit == 3 {
		s.res_name = RESOURCE_NAME_1080P
		s.socket_name = SOLIOS_1080P_SOCKET
	} else if s.allocation_unit == 4 {
		s.res_name = RESOURCE_NAME_2160P
		s.socket_name = SOLIOS_2160P_SOCKET
	}

	C.srm_init()

	count, err := s.listDevice()
	if err != nil {
		log.Fatalf("list device error: %v, count:%v", count, err)
	}

	pluginapi.RegisterDevicePluginServer(s.srv, s)
	err = syscall.Unlink(DEVICE_PLUG_PATH + s.socket_name)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	l, err := net.Listen("unix", DEVICE_PLUG_PATH+s.socket_name)
	if err != nil {
		return err
	}

	go func() {
		lastCrashTime := time.Now()
		restartCount := 0
		for {
			log.Printf("start GPPC server for '%s'", s.res_name)
			err = s.srv.Serve(l)
			if err == nil {
				break
			}

			log.Printf("GRPC server for '%s' crashed with error: $v", s.res_name, err)

			if restartCount > 5 {
				log.Fatal("GRPC server for '%s' has repeatedly crashed recently. Quitting", s.res_name)
			}
			timeSinceLastCrash := time.Since(lastCrashTime).Seconds()
			lastCrashTime = time.Now()
			if timeSinceLastCrash > 3600 {
				restartCount = 1
			} else {
				restartCount++
			}
		}
	}()

	// Wait for server to start by lauching a blocking connection
	conn, err := s.dial(s.socket_name, 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()

	return nil
}

func (s *SoliosServer) listDevice() (int, error) {
	count := 0

	if s.allocation_unit == 0 {
		dir, err := ioutil.ReadDir(LOCATION)
		if err != nil {
			return 0, err
		}
		for _, f := range dir {
			if f.IsDir() {
				continue
			}
			if strings.HasPrefix(f.Name(), SOLIOS_DEVICE_PREFIX) {
				s.devices[f.Name()] = &pluginapi.Device{
					ID:     f.Name(),
					Health: pluginapi.Healthy,
				}
				count++
				log.Infof("found device '%s'", f.Name())
			}
		}
	} else {
		cress := C.srm_get_total_resource(C.int(s.allocation_unit))
		for i := 0; i < int(cress); i++ {
			s.devices[strconv.Itoa(i)] = &pluginapi.Device{
				ID:     strconv.Itoa(i),
				Health: pluginapi.Healthy,
			}
		}
		count = int(cress)
	}
	log.Infof("allocation_unit=%d, resources '%s' = %d", s.allocation_unit, s.res_name, count)
	return count, nil
}

func (s *SoliosServer) ListAndWatch(e *pluginapi.Empty, srv pluginapi.DevicePlugin_ListAndWatchServer) error {
	log.Infoln("ListAndWatch called")

	devs := make([]*pluginapi.Device, len(s.devices))
	i := 0
	for i < len(s.devices) {
		devs[i] = &pluginapi.Device{
			ID:     strconv.Itoa(i),
			Health: pluginapi.Healthy,
		}
		i++
	}
	srv.Send(&pluginapi.ListAndWatchResponse{Devices: devs})

	//device mode
	if s.allocation_unit == 0 {
		for {
			log.Infoln("waiting for device change")
			select {
			case <-s.notify:
				log.Infoln("start to update device list, devices: %d", len(s.devices))
				devs := make([]*pluginapi.Device, len(s.devices))

				i := 0
				for _, dev := range s.devices {
					devs[i] = dev
					i++
				}
				srv.Send(&pluginapi.ListAndWatchResponse{Devices: devs})

			case <-s.ctx.Done():
				log.Info("ListAndWatch exit")
				return nil
			}
		}
	} else { //480p/720p/1080p/2160p mode
		ticker := time.NewTicker(time.Second * 5)
		for range ticker.C {
			count, err := s.listDevice()
			if err != nil {
				return fmt.Errorf("listDevice error:%v", err)
			}
			devs := make([]*pluginapi.Device, count)
			i := 0
			for i < count {
				devs[i] = &pluginapi.Device{
					ID:     strconv.Itoa(i),
					Health: pluginapi.Healthy,
				}
				i++
			}
			srv.Send(&pluginapi.ListAndWatchResponse{Devices: devs})
		}
	}

	log.Info("ListAndWatch exit")
	return nil
}

func (s *SoliosServer) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resps := &pluginapi.AllocateResponse{}
	var count, driver_id int = 0, 0

	log.Infoln("Allocate was called")

	if s.allocation_unit == 0 {
		for _, req := range reqs.ContainerRequests {
			log.Infof("received request: devices = %v", strings.Join(req.DevicesIDs, ","))
			rsp := pluginapi.ContainerAllocateResponse{}
			for _, devname := range req.DevicesIDs {
				path := "/dev/transcoder" + devname
				rsp.Devices = append(rsp.Devices, &pluginapi.DeviceSpec{
					HostPath:      path,
					ContainerPath: path,
					Permissions:   "rwm",
				})
				log.Infof("Added device: %v", path)
			}
			resps.ContainerResponses = append(resps.ContainerResponses, &rsp)
		}
	} else {
		for _, req := range reqs.ContainerRequests {
			count += len(req.DevicesIDs)
		}

		log.Infof("Resource required: %v", count)
		driver_id = int(C.srm_allocate_resource(C.int(s.power_mode), C.int(s.allocation_unit), C.int(count)))

		if driver_id == -1 {
			log.Infof("Can't Matched any device")
			return resps, nil
		}

		rsp := pluginapi.ContainerAllocateResponse{}
		path := "/dev/transcoder" + strconv.Itoa(driver_id)
		rsp.Devices = append(rsp.Devices, &pluginapi.DeviceSpec{
			HostPath:      path,
			ContainerPath: path,
			Permissions:   "rwm",
		})
		log.Infof("Added device: %v", path)
		resps.ContainerResponses = append(resps.ContainerResponses, &rsp)
	}
	return resps, nil
}

func (s *SoliosServer) watchDevice() error {
	log.Infoln("watching devices")
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("NewWatcher error:%v", err)
	}
	defer w.Close()

	if s.allocation_unit > 0 {
		log.Info("allocation_unit is %d, exit watchDevice", s.allocation_unit)
		return nil
	}

	done := make(chan bool)
	go func() {
		defer func() {
			done <- true
			log.Info("watch device exit")
		}()
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					continue
				}

				if event.Op&fsnotify.Create == fsnotify.Create {
					if !strings.HasPrefix(event.Name, SOLIOS_DEVICE_PREFIX) {
						continue
					}

					s.devices[event.Name] = &pluginapi.Device{
						ID:     event.Name,
						Health: pluginapi.Healthy,
					}
					s.notify <- true
					log.Infoln("new device find:", event.Name)
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {

					delete(s.devices, event.Name)
					s.notify <- true
					log.Infoln("device deleted:", event.Name)
				}

			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)

			case <-s.ctx.Done():
				break
			}
		}
	}()

	err = w.Add(LOCATION)
	if err != nil {
		return fmt.Errorf("watch device error:%v", err)
	}
	<-done

	return nil
}

func (s *SoliosServer) PreStartContainer(ctx context.Context, req *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	log.Infoln("PreStartContainer called")
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (s *SoliosServer) dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	c, err := grpc.Dial(unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func (s *SoliosServer) RegisterToKubelet() error {
	socketFile := filepath.Join(DEVICE_PLUG_PATH + KUBELET_SOCKET)

	conn, err := s.dial(socketFile, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	req := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(DEVICE_PLUG_PATH + s.socket_name),
		ResourceName: s.res_name,
	}

	log.Infof("Register to kubelet with endpoint %s", req.Endpoint)
	_, err = client.Register(context.Background(), req)
	if err != nil {
		return err
	}
	return nil
}

func (s *SoliosServer) GetDevicePluginOptions(ctx context.Context, e *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	log.Infoln("GetDevicePluginOptions called")
	return &pluginapi.DevicePluginOptions{PreStartRequired: true}, nil
}
