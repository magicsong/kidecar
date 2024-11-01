package httpprobe

import "github.com/magicsong/kidecar/pkg/store"

type EndpointConfig struct {
	URL                string                `json:"url"`                // 目标 URL
	Method             string                `json:"method"`             // HTTP 方法
	Headers            map[string]string     `json:"headers"`            // 请求头
	Timeout            int                   `json:"timeout"`            // 超时时间（秒）
	ExpectedStatusCode int                   `json:"expectedStatusCode"` // 预期的 HTTP 状态码
	StorageConfig      store.StorageConfig   `json:"storageConfig"`      // 存储配置
	JSONPathConfig     *store.JSONPathConfig `json:"jsonPathConfig"`     // JSONPath 配置
}

type HttpProbeConfig struct {
	StartDelaySeconds    int              `json:"startDelaySeconds"`    // 延迟启动时间（秒）
	Endpoints            []EndpointConfig `json:"endpoints,omitempty"`  // 多个端点的配置
	ProbeIntervalSeconds int              `json:"probeIntervalSeconds"` // 探测间隔时间（秒）
}
