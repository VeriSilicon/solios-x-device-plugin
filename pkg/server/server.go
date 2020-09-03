package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	RESOURCE_NAME        string = "verisilicon.com/solios"
	LOCATION             string = "/dev"
	SOLIOS_SOCKET        string = "solios.sock"
	SOLIOS_DEVICE_PREFIX string = "transcoder"
	KUBELET_SOCKET       string = "kubelet.sock"
	DEVICE_PLUG_PATH     string = "/var/lib/kubelet/device-plugins/"
)

type SoliosServer struct {
	srv         *grpc.Server
	devices     map[string]*pluginapi.Device
	notify      chan bool
	ctx         context.Context
	cancel      context.CancelFunc
	restartFlag bool //restart?
}

func NewSoliosServer() *SoliosServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &SoliosServer{
		devices:     make(map[string]*pluginapi.Device),
		srv:         grpc.NewServer(grpc.EmptyServerOption{}),
		notify:      make(chan bool),
		ctx:         ctx,
		cancel:      cancel,
		restartFlag: false,
	}
}

func (s *SoliosServer) Run() error {
	err := s.listDevice()
	if err != nil {
		log.Fatalf("list device error: %v", err)
	}

	go func() {
		err := s.watchDevice()
		if err != nil {
			log.Println("watch devices error")
		}
	}()

	pluginapi.RegisterDevicePluginServer(s.srv, s)
	err = syscall.Unlink(DEVICE_PLUG_PATH + SOLIOS_SOCKET)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	l, err := net.Listen("unix", DEVICE_PLUG_PATH+SOLIOS_SOCKET)
	if err != nil {
		return err
	}

	go func() {
		lastCrashTime := time.Now()
		restartCount := 0
		for {
			log.Printf("start GPPC server for '%s'", RESOURCE_NAME)
			err = s.srv.Serve(l)
			if err == nil {
				break
			}

			log.Printf("GRPC server for '%s' crashed with error: $v", RESOURCE_NAME, err)

			if restartCount > 5 {
				log.Fatal("GRPC server for '%s' has repeatedly crashed recently. Quitting", RESOURCE_NAME)
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
	conn, err := s.dial(SOLIOS_SOCKET, 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()

	return nil
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
		Endpoint:     path.Base(DEVICE_PLUG_PATH + SOLIOS_SOCKET),
		ResourceName: RESOURCE_NAME,
	}
	log.Infof("Register to kubelet with endpoint %s", req.Endpoint)
	_, err = client.Register(context.Background(), req)
	if err != nil {
		return err
	}

	return nil
}

// GetDevicePluginOptions returns options to be communicated with Device
// Manager
func (s *SoliosServer) GetDevicePluginOptions(ctx context.Context, e *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	log.Infoln("GetDevicePluginOptions called")
	return &pluginapi.DevicePluginOptions{PreStartRequired: true}, nil
}

// ListAndWatch returns a stream of List of Devices
// Whenever a Device state change or a Device disappears, ListAndWatch
// returns the new list
func (s *SoliosServer) ListAndWatch(e *pluginapi.Empty, srv pluginapi.DevicePlugin_ListAndWatchServer) error {
	log.Infoln("ListAndWatch called")
	devs := make([]*pluginapi.Device, len(s.devices))

	i := 0
	for _, dev := range s.devices {
		devs[i] = dev
		i++
	}

	err := srv.Send(&pluginapi.ListAndWatchResponse{Devices: devs})
	if err != nil {
		log.Errorf("ListAndWatch send device error: %v", err)
		return err
	}

	// update device list
	for {
		log.Infoln("waiting for device change")
		select {
		case <-s.notify:
			log.Infoln("start to uopdate device list, devices:", len(s.devices))
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
}

func (s *SoliosServer) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	log.Infoln("Allocate was called")
	resps := &pluginapi.AllocateResponse{}

	for _, req := range reqs.ContainerRequests {
		log.Infof("received request: %v", strings.Join(req.DevicesIDs, ","))
		rsp := pluginapi.ContainerAllocateResponse{}
		for _, devname := range req.DevicesIDs {
			rsp.Devices = append(rsp.Devices, &pluginapi.DeviceSpec{
				HostPath:      "/dev/" + devname,
				ContainerPath: "/dev/" + devname,
				Permissions:   "rwm",
			})
			log.Infof("Added device: %v", devname)
		}
		resps.ContainerResponses = append(resps.ContainerResponses, &rsp)
	}
	return resps, nil
}

func (s *SoliosServer) PreStartContainer(ctx context.Context, req *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	log.Infoln("PreStartContainer called")
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (s *SoliosServer) listDevice() error {
	dir, err := ioutil.ReadDir(LOCATION)
	if err != nil {
		return err
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
			log.Infof("found device '%s'", f.Name())
		}
	}
	return nil
}

func (s *SoliosServer) watchDevice() error {
	log.Infoln("watching devices")
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("NewWatcher error:%v", err)
	}
	defer w.Close()

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
				log.Infoln("device event:", event.Op.String())

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
