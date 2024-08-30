package assembler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/magicsong/okg-sidecar/api"
	"github.com/magicsong/okg-sidecar/pkg/utils"
	"gopkg.in/yaml.v3"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var _ api.Sidecar = &sidecar{}

type sidecar struct {
	plugins          map[string]api.Plugin
	lock             sync.RWMutex
	isStartWebServer bool
	version          string
	pluginStatuses   map[string]*api.PluginStatus
	*api.SidecarConfig
	api.SidecarManager
	log logr.Logger
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
func (s *sidecar) SetupWithManager(mgr api.SidecarManager) error {
	s.SidecarManager = mgr
	return nil
}

func NewSidecar() api.Sidecar {
	return &sidecar{
		plugins:        make(map[string]api.Plugin),
		pluginStatuses: make(map[string]*api.PluginStatus),
		log:            logf.Log.WithName("sidecar"),
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
	pluginConfig := plugin.GetConfigType()
	pluginOption, ok := s.SidecarConfig.Plugins[plugin.Name()]
	if !ok {
		s.log.Info("plugin not found in config, skip", "plugin", plugin.Name())
		return nil
	}
	err := utils.ConvertJsonObjectToStruct(pluginOption.Config, pluginConfig)
	if err != nil {
		return fmt.Errorf("convert plugin config failed,err:%w", err)
	}
	if err := plugin.Init(pluginConfig, s.SidecarManager); err != nil {
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
	s.log.Info("start polling plugin status", "plugin", pluginName)
	defer s.log.Info("end polling plugin status", "plugin", pluginName)
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
	s.log.Info("start sidecar")
	errorCh := make(chan error)
	s.startAllPlugins(ctx, errorCh)
	for _, plugin := range s.plugins {
		s.pollPluginStatus(plugin.Name(), time.Second*5)
		time.Sleep(time.Second)
	}
	if s.isStartWebServer {
		// start server
		go s.startServer()
	}
	s.log.Info("sidecar started successfully")
	// wait for error
	err := <-errorCh
	s.log.Error(err, "plugin error")
	return err
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

func (s *sidecar) isPluginEnabled(pluginName string) bool {
	pluginOption, ok := s.SidecarConfig.Plugins[pluginName]
	if !ok {
		return true
	}
	return pluginOption.BootOrder > 0
}

func (s *sidecar) startAllPlugins(ctx context.Context, errorCh chan<- error) {
	for _, plugin := range s.plugins {
		s.log.Info("start plugin", "plugin", plugin.Name())
		if s.isPluginRunning(plugin.Name()) {
			continue
		}
		go plugin.Start(ctx, errorCh)
		s.log.Info("plugin started successfully", "plugin", plugin.Name())
	}
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
