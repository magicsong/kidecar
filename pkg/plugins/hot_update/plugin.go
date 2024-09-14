package hot_update

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/magicsong/okg-sidecar/api"
	"github.com/magicsong/okg-sidecar/pkg/store"
	"net/http"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type hotUpdate struct {
	config HotUpdateConfig
	store.StorageFactory
	status *HotUpdateStatus
	result *HotUpdateResult
	log    logr.Logger
}

type HotUpdateResult struct {
	Result  string `json:"result"`
	Version string `json:"version"`
	Url     string `json:"url"`
}

type HotUpdateConfig struct {
	LoadPatchType string              `json:"loadPatchType"`
	Signal        Signal              `json:"signal,omitempty"`
	Request       Request             `json:"request,omitempty"`
	FileDir       string              `json:"fileDir"`
	StorageConfig store.StorageConfig `json:"storageConfig,omitempty"`
}

// Signal 发送信号量到主容器
type Signal struct {
	SignalName  string `json:"signalName"`
	ProcessName string `json:"processName"`
}

type Request struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

func (h *hotUpdate) Name() string {
	return pluginName
}

func (h *hotUpdate) Init(config interface{}, mgr api.SidecarManager) error {
	hotUpdateConfig, ok := config.(*HotUpdateConfig)
	if !ok {
		return fmt.Errorf("invalid config type of hot-update, config: %v", config)
	}
	err := validateConfig(hotUpdateConfig)
	if err != nil {
		return fmt.Errorf("invalid config of hot-update, err: %v", err)
	}

	h.config = *hotUpdateConfig
	h.status = &HotUpdateStatus{}
	h.result = &HotUpdateResult{}
	h.StorageFactory = store.NewStorageFactory(mgr)
	h.log = logf.Log.WithName("hot-update")
	return nil
}

func (h *hotUpdate) Start(ctx context.Context, errCh chan<- error) {
	h.log.Info("start hot-update plugin")

	// 根据pod的anno，设置最新的热更新文件
	err := h.setHotUpdateConfigWhenStart()
	if err != nil {
		h.log.Error(err, "Failed to set hot-update config when start")
		h.status.setStatus("Stopped")
		errCh <- err
		return
	}

	// 启动一个http服务
	http.HandleFunc("/hot-update", h.hotUpdateHandle)

	err = http.ListenAndServe(":5000", nil)
	if err != nil {
		h.log.Error(err, "Failed to start hot-update plugin")
		h.status.setStatus("Stopped")
		errCh <- err
		return
	}
	h.status.setStatus("Running")
}

func (h *hotUpdate) Stop(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (h *hotUpdate) Version() string {
	return "v0.0.1"
}

func (h *hotUpdate) Status() (*api.PluginStatus, error) {
	return &api.PluginStatus{
		Name:    pluginName,
		Health:  h.status.getStatus(),
		Running: h.status.getStatus() == "Running",
	}, nil
}

func (h *hotUpdate) GetConfigType() interface{} {
	return &HotUpdateConfig{}
}

func NewPlugin() api.Plugin {
	return &hotUpdate{}
}
