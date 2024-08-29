package assembler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/magicsong/okg-sidecar/api"
	"github.com/magicsong/okg-sidecar/pkg/manager"
	"github.com/magicsong/okg-sidecar/pkg/utils"
	"gopkg.in/yaml.v2"
)

var _ api.Sidecar = &sidecar{}

type sidecar struct {
	plugins          map[string]api.Plugin
	lock             sync.RWMutex
	isStartWebServer bool
	version          string
	pluginStatuses   map[string]*api.PluginStatus
	*api.SidecarConfig
	*manager.SidecarManager
}

// LoadConfig implements api.Sidecar.
func (s *sidecar) LoadConfig(path string) error {
	config, err := loadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to load config from path %s, err: %w", path, err)
	}
	s.SidecarConfig = config
	return nil
}

func loadConfig(configPath string) (*api.SidecarConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	sidecarConfig := &api.SidecarConfig{}
	if err := yaml.Unmarshal(data, &sidecarConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sidecarConfig file: %w", err)
	}
	return sidecarConfig, nil
}

// SetupWithManager implements api.Sidecar.
func (s *sidecar) SetupWithManager(mgr *manager.SidecarManager) error {
	s.SidecarManager = mgr
	return nil
}

func NewSidecar() api.Sidecar {
	return &sidecar{
		plugins:        make(map[string]api.Plugin),
		pluginStatuses: make(map[string]*api.PluginStatus),
	}
}

// AddPlugin implements api.Sidecar.
func (s *sidecar) AddPlugin(plugin api.Plugin) error {
	//lock and add
	s.lock.Lock()
	defer s.lock.Unlock()
	if plugin.Name() == "" {
		return fmt.Errorf("plugin name is empty")
	}
	var pluginConfig interface{}
	pluginOption, ok := s.SidecarConfig.Plugins[plugin.Name()]
	if !ok {
		pluginConfig = plugin.GetConfigType()
	} else {
		err := utils.ConvertJsonObjectToStruct(pluginOption.Config, plugin.GetConfigType())
		if err != nil {
			return fmt.Errorf("convert plugin config failed")
		}
	}
	if err := plugin.Init(pluginConfig); err != nil {
		return fmt.Errorf("init plugin %s failed", plugin.Name())
	}
	s.plugins[plugin.Name()] = plugin
	return nil
}

// GetVersion implements api.Sidecar.
func (s *sidecar) GetVersion() string {
	// get version from git
	return s.version
}

// PluginStatus implements api.Sidecar.
func (s *sidecar) PluginStatus(pluginName string) (*api.PluginStatus, error) {
	if status, ok := s.pluginStatuses[pluginName]; ok {
		return status, nil
	}
	return s.updatePluginStatus(pluginName)
}

func (s *sidecar) updatePluginStatus(pluginName string) (*api.PluginStatus, error) {

	plugin, ok := s.plugins[pluginName]
	if !ok {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}
	status, err := plugin.Status()
	if err != nil {
		return nil, fmt.Errorf("get plugin %s status failed", pluginName)
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.pluginStatuses[pluginName] = status
	return status, nil
}

// RemovePlugin implements api.Sidecar.
func (s *sidecar) RemovePlugin(pluginName string) error {
	//lock and remove
	s.lock.Lock()
	defer s.lock.Unlock()
	if plugin, ok := s.plugins[pluginName]; ok {
		if err := plugin.Stop(context.Background()); err != nil {
			return fmt.Errorf("stop plugin %s failed", pluginName)
		}
		delete(s.plugins, pluginName)
		return nil
	}
	return fmt.Errorf("plugin %s not found", pluginName)
}

// Start implements api.Sidecar.
func (s *sidecar) Start(ctx context.Context) error {
	// start all plugins
	s.lock.RLock()
	defer s.lock.RUnlock()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	if err := s.StartAllPlugins(ctxWithTimeout); err != nil {
		return fmt.Errorf("start all plugins failed")
	}
	for _, plugin := range s.plugins {
		s.pollPluginStatus(plugin.Name(), time.Second*5)
		time.Sleep(time.Second)
	}
	if s.isStartWebServer {
		// start server
		go s.startServer()
	}
	return nil
}

func (s *sidecar) startServer() {
	// start server
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("start web server failed: %v", err)
	}
}

// pollPluginStatus periodically polls the status of a plugin with the given time interval.
func (s *sidecar) pollPluginStatus(pluginName string, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			s.updatePluginStatus(pluginName)
		}
	}()
}

// StartAllPlugins implements api.Sidecar.
func (s *sidecar) StartAllPlugins(ctx context.Context) error {
	for _, plugin := range s.plugins {
		if s.isPluginRunning(plugin.Name()) {
			return nil
		}
		if err := plugin.Start(ctx); err != nil {
			return fmt.Errorf("start plugin %s failed", plugin.Name())
		}
	}
	return nil
}

// Stop implements api.Sidecar.
func (s *sidecar) Stop(ctx context.Context) error {
	// stop all plugins
	s.lock.RLock()
	defer s.lock.RUnlock()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	if err := s.StopAllPlugins(ctxWithTimeout); err != nil {
		return fmt.Errorf("stop all plugins failed")
	}
	return nil
}

// StopAllPlugins implements api.Sidecar.
func (s *sidecar) StopAllPlugins(ctx context.Context) error {
	panic("unimplemented")
}

func (s *sidecar) isPluginRunning(pluginName string) bool {
	status, err := s.PluginStatus(pluginName)
	if err != nil {
		return false
	}
	return status.Running
}
