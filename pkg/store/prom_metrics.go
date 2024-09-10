package store

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/magicsong/okg-sidecar/api"
	"github.com/magicsong/okg-sidecar/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type promMetric struct {
	registry  *prometheus.Registry
	metrics   map[string]prometheus.Gauge
	metricsMu sync.Mutex
}

// IsInitialized implements Storage.
func (p *promMetric) IsInitialized() bool {
	return p.registry != nil
}

// SetupWithManager implements Storage.
func (p *promMetric) SetupWithManager(mgr api.SidecarManager) error {
	reg := prometheus.NewRegistry()
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// 启动HTTP服务器
	fmt.Println("Starting server at :8080")
	p.registry = reg
	go http.ListenAndServe(":8080", nil)
	return nil
}

// Store implements Storage.
func (p *promMetric) Store(data interface{}, config interface{}) error {
	myconfig, ok := config.(*HTTPMetricConfig)
	if !ok {
		return fmt.Errorf("bad config of httpMetricConfig")
	}
	f, err := strconv.ParseFloat(utils.ConvertToString(data), 64)
	if err != nil {
		return fmt.Errorf("bad data of httpMetricConfig, err: %w", err)
	}
	p.getOrCreateGauge(myconfig).Set(f)
	return nil
}

// getOrCreateGauge 获取现有的Gauge或者创建一个新的
func (p *promMetric) getOrCreateGauge(config *HTTPMetricConfig) prometheus.Gauge {
	p.metricsMu.Lock()
	defer p.metricsMu.Unlock()

	if gauge, exists := p.metrics[config.MetricName]; exists {
		return gauge
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: config.MetricName,
		Help: "Automatically generated metric from collected data",
	})
	p.registry.MustRegister(gauge)
	p.metrics[config.MetricName] = gauge
	return gauge
}
