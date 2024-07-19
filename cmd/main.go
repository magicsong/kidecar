package main

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	ctrlLog "sigs.k8s.io/controller-runtime/pkg/log"
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

var sc = config.NewSafeConfig(prometheus.DefaultRegisterer)

func main() {
	// Load Kubernetes configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	// Create Kubernetes clientset
	clientset := kubernetes.NewForConfigOrDie(config)
	okgClientset := kruisegameclientset.NewForConfigOrDie(config)

	gamePatch := pkg.NewGamePatcher(clientset, okgClientset)
	// Get current GameServerSet information
	gssName := os.Getenv("GAME_SERVER_SET_NAME")
	namespace := os.Getenv("GAME_SERVER_SET_NAMESPACE")

	// Create logger
	logger := log.NewNopLogger()

	if err = sc.ReloadConfig("blackbox.yml", logger); err != nil {
		ctrlLog.FromContext(context.Background()).Error(err, "Error loading blackbox config")
		panic(err.Error())
	}
	ctrlLog.Log.Info("blackbox config loaded, Modules: ", sc.C.Modules)

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
						ctrlLog.FromContext(context.Background()).Error(err, "Error loading blackbox config")
					}
					ctrlLog.Log.Info("blackbox config loaded, Modules: ", sc.C.Modules)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				ctrlLog.FromContext(context.Background()).Error(err, "Error watching config file")
			}
		}
	}()

	err = watcher.Add("blackbox.yml")
	if err != nil {
		panic(err.Error())
	}

	// Start monitoring loop
	for {
		// Perform HTTP probe using Blackbox Exporter
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// Get GameServer by gameserverset selector
		gsList, err := gamePatch.ListGameServerByGSS(ctx, gssName, namespace)
		if err != nil && errors.IsNotFound(err) {
			ctrlLog.FromContext(ctx).Info("GameServer not found by gss", " gssName: ", gssName, " namespace: ", namespace)
			continue
		}
		ctrlLog.Log.Info("GameServer Info ", " len(gs): ", len(gsList.Items))

		registry := prometheus.DefaultRegisterer.(*prometheus.Registry)
		for _, gs := range gsList.Items {
			// Probe GameServer Port By HTTP
			ctrlLog.Log.Info("GameServer Probe", "gssName:", gs.Name, " gssNamespace:", gs.Namespace)
			// Get HTTP RUL by gs
			if url, ok := gs.Annotations["http-url"]; ok {
				success := prober.ProbeHTTP(ctx, url, sc.C.Modules["http_2xx"], registry, logger)
				ctrlLog.Log.Info("GameServer Probe", "gssName:", gs.Name, " gssNamespace:", gs.Namespace, "success:", success)
				if success && gs.Spec.OpsState != "Allocated" {
					gs.Spec.OpsState = "Allocated"
					if err = gamePatch.PatchGameServer(ctx, &gs); err != nil {
						fmt.Printf("Error patching GameServer status: %v\n", err)
					}
				} else if !success && gs.Spec.OpsState != "WaitToBeDeleted" {
					gs.Spec.OpsState = "WaitToBeDeleted"
					if err = gamePatch.PatchGameServer(ctx, &gs); err != nil {
						fmt.Printf("Error patching GameServer status: %v\n", err)
					}
				}
			}

		}

		// Sleep for a while before next iteration
		time.Sleep(100 * time.Second)
	}
}
