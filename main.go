package main

import (
	"os"
	"path/filepath"
	"time"

	solios "github.com/VeriSilicon/solios/pkg/server"
	log "github.com/sirupsen/logrus"
	"gopkg.in/fsnotify.v1"
)

func main() {
	log.Info("solios device plugin starting")
	soliosSrv := solios.NewSoliosServer()
	go soliosSrv.Run()

	if err := soliosSrv.RegisterToKubelet(); err != nil {
		log.Fatalf("register to kubelet error: %v", err)
	} else {
		log.Infoln("register to kubelet successfully")
	}

	devicePluginSocket := filepath.Join(solios.DEVICE_PLUG_PATH, solios.KUBELET_SOCKET)
	log.Info("device plugin socket name:", devicePluginSocket)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error("Failed to created FS watcher.")
		os.Exit(1)
	}
	defer watcher.Close()
	err = watcher.Add(solios.DEVICE_PLUG_PATH)
	if err != nil {
		log.Error("watch kubelet error")
		return
	}
	log.Info("watching kubelet.sock")
	for {
		select {
		case event := <-watcher.Events:
			if event.Name == devicePluginSocket && event.Op&fsnotify.Create == fsnotify.Create {
				time.Sleep(time.Second)
				log.Fatalf("inotify: %s created, restarting.", devicePluginSocket)
			}
		case err := <-watcher.Errors:
			log.Fatalf("inotify: %s", err)
		}
	}
}
