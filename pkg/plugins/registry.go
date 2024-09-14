package plugins

import (
	"github.com/magicsong/okg-sidecar/api"
	"github.com/magicsong/okg-sidecar/pkg/plugins/hot_update"
	httpprobe "github.com/magicsong/okg-sidecar/pkg/plugins/http_probe"
)

var PluginRegistry = make(map[string]api.Plugin)

func RegisterPlugin(plugin api.Plugin) {
	if plugin.Name() == "" {
		panic("plugin name is empty")
	}
	PluginRegistry[plugin.Name()] = plugin
}

func init() {
	// 注册plugin
	RegisterPlugin(httpprobe.NewPlugin())
	RegisterPlugin(hot_update.NewPlugin())
}
