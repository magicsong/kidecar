package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-kit/log"
	"github.com/magicsong/okg-sidecar/pkg"
	kruisegameclientset "github.com/openkruise/kruise-game/pkg/client/clientset/versioned"
	"github.com/prometheus/blackbox_exporter/config"
	"github.com/prometheus/blackbox_exporter/prober"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type GameServerStatus struct {
	IsIdle           bool   `json:"isIdle"`
	IsMaintaining    bool   `json:"isMaintaining"`
	CanAcceptPlayers bool   `json:"canAcceptPlayers"`
	PlayerCount      int    `json:"playerCount"`
	Url              string `json:"url"`
}

var sc = config.NewSafeConfig(prometheus.DefaultRegisterer)

func main() {
	// Load Kubernetes configuration
	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Create Kubernetes clientset
	clientset := kubernetes.NewForConfigOrDie(config)
	okgClientset := kruisegameclientset.NewForConfigOrDie(config)

	gamePatch := pkg.NewGamePatcher(clientset, okgClientset)
	// Get current Pod information
	podName := os.Getenv("POD_NAME")
	namespace := os.Getenv("POD_NAMESPACE")

	// Create logger
	logger := log.NewNopLogger()

	if err = sc.ReloadConfig("blackbox.yml", logger); err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		panic(err.Error())
	}

	// Set up file watcher for config file
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err.Error())
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					err := sc.ReloadConfig("blackbox.yml", logger)
					if err != nil {
						fmt.Printf("Error reloading config: %v\n", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("Error watching config file: %v\n", err)
			}
		}
	}()

	err = watcher.Add("blackbox.yml")
	if err != nil {
		panic(err.Error())
	}

	// Start monitoring loop
	for {
		// Get GameServer information
		var gameServerStatus GameServerStatus

		// Perform HTTP probe using Blackbox Exporter
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		registry := prometheus.DefaultRegisterer.(*prometheus.Registry)

		success := prober.ProbeHTTP(ctx, gameServerStatus.Url, sc.C.Modules["http_2xx"], registry, logger)
		if success {
			if err = gamePatch.PatchGameServerStatus(ctx, podName, namespace, "Ready"); err != nil {
				fmt.Printf("Error patching GameServer status: %v\n", err)
			}
		}
		// Sleep for a while before next iteration
		time.Sleep(10 * time.Second)
	}
}