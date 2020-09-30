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
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	solios "github.com/VeriSilicon/solios/pkg/server"
	log "github.com/sirupsen/logrus"
	"gopkg.in/fsnotify.v1"
)

func main() {

	var allocation_unit = flag.String("allocation_unit", "", "resource allocation unit, can be solios/480p/720p/1080p/2160p")
	var power_mode = flag.String("power_mode", "", "resource allocation mode. can be power_saving/balance")

	log.Info("solios device plugin starting")
	flag.Parse()
	fmt.Println("-allocation_unit", *allocation_unit)
	fmt.Println("-power_mode", *power_mode)

	soliosSrv := solios.NewSoliosServer()
	go soliosSrv.Run(*allocation_unit, *power_mode)

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
