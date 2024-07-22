package main

import (
	"context"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/homedir"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/magicsong/okg-sidecar/pkg"
	kruisegameclientset "github.com/openkruise/kruise-game/pkg/client/clientset/versioned"
	"github.com/prometheus/blackbox_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var sc = config.NewSafeConfig(prometheus.DefaultRegisterer)

func main() {
	// Load Kubernetes configuration
	config, err := rest.InClusterConfig()
	klog.Info("InClusterConfig")
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

	// Start monitoring loop
	for {
		// Perform HTTP probe using Blackbox Exporter
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// Get GameServer by probe selector
		gsList, err := gamePatch.ListGameServersByProbeLabel(ctx)
		if (err != nil && errors.IsNotFound(err)) || gsList == nil {
			klog.FromContext(ctx).Error(err, "GameServer not found")
			time.Sleep(10 * time.Second)
			continue
		}

		klog.Info("GameServer Info ", " len(gs): ", len(gsList.Items))
		for _, gs := range gsList.Items {
			// Probe GameServer Port By HTTP
			klog.Info("GameServer Probe", "gssName:", gs.Name, " gssNamespace:", gs.Namespace)
			// Get HTTP RUL by gs
			podIp := gs.Status.PodStatus.PodIP
			podPort := gs.Labels[pkg.GMAESERVER_PROBE_PORT_LABEL]
			wantResp := gs.Labels[pkg.GAMESERVER_WANT_RESP]
			klog.Info("wantResp: ", wantResp)

			if podPort != "" && wantResp != "" {
				url := "http://" + podIp + ":" + podPort
				err, success := sendHTTP(url, wantResp)
				if err != nil {
					klog.Error(err, "falied to send http", " url is ", url)
				}
				klog.Info("GameServer Probe", " gsName:", gs.Name, " gsNamespace:", gs.Namespace, " url", url, " success:", success)
				if success && gs.Spec.OpsState != "Allocated" {
					gs.Spec.OpsState = "Allocated"
					if err = gamePatch.PatchGameServer(ctx, &gs); err != nil {
						klog.Error("Error patching GameServer status: %v\n", err)
					}
				} else if !success && gs.Spec.OpsState == "Allocated" {
					gs.Spec.OpsState = "WaitToBeDeleted"
					if err = gamePatch.PatchGameServer(ctx, &gs); err != nil {
						klog.Error("Error patching GameServer status: %v\n", err)
					}
				}
			}

		}

		// Sleep for a while before next iteration
		time.Sleep(10 * time.Second)
	}
}

func sendHTTP(url, wantStr string) (error, bool) {
	resp, err := http.Get(url)
	if err != nil {
		klog.Error(err, "http get error")
		return err, false
	}
	defer resp.Body.Close()

	klog.Info("http status code is: ", resp.StatusCode)
	// 检查HTTP响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}

	// 读取响应的body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Error(err, "read http body error")
		return err, false
	}

	// 将响应的body转为字符串
	bodyString := string(body)

	// 检查响应内容中是否包含"idle"
	klog.Info("http bodyString: ", bodyString)
	if strings.Contains(bodyString, wantStr) {
		return nil, true
	} else {
		return nil, false
	}
}
