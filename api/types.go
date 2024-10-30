package api

import (
	"context"

	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Plugin 定义所有插件必须实现的方法
type Plugin interface {
	Name() string
	Init(config interface{}, mgr SidecarManager) error
	Start(ctx context.Context, errCh chan<- error)
	Stop(ctx context.Context) error
	Version() string
	Status() (*PluginStatus, error)
	// must return a pointer of your config
	GetConfigType() interface{}
}

// PluginConfig 表示插件的配置
type PluginConfig struct {
	Name      string      `json:"name"`
	Config    interface{} `json:"config"`
	BootOrder int         `json:"bootOrder"`
}

// SidecarConfig 表示 Sidecar 的配置
type SidecarConfig struct {
	Plugins           []PluginConfig    `json:"plugins"`           // 启动的插件及其配置
	RestartPolicy     string            `json:"restartPolicy"`     // 重启策略
	Resources         map[string]string `json:"resources"`         // Sidecar 所需的资源
	SidecarStartOrder string            `json:"sidecarStartOrder"` // Sidecar 的启动顺序，是在主容器之后还是之前
}

// PluginStatus 表示插件的状态
type PluginStatus struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Running     bool     `json:"running"`
	LastChecked string   `json:"lastChecked"` // 上一次健康检查时间，格式为 YYYY-MM-DD HH:MM:SS
	Health      string   `json:"health"`      // 健康状态，例如 "Healthy", "Unhealthy"
	Infos       []string `json:"infos"`       // 插件的其他信息
}

// Sidecar 定义Sidecar对外的接口
type Sidecar interface {
	AddPlugin(plugin Plugin) error
	RemovePlugin(pluginName string) error
	GetVersion() string
	PluginStatus(pluginName string) (*PluginStatus, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SetupWithManager(mgr SidecarManager) error
	LoadConfig(path string) error
}

type SidecarManager interface {
	ctrl.Manager
	DBManager
	kubernetes.Interface
}

type DBManager interface {
}
