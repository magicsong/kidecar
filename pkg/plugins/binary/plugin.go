package binary

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/magicsong/kidecar/api"
	"github.com/magicsong/kidecar/api/v1alpha1"
)

func NewPlugin() api.Plugin {
	return &binary{
		name: "binary",
	}
}

type binary struct {
	name   string
	config v1alpha1.Binary

	cmd    *exec.Cmd
	status *api.PluginStatus
	mu     sync.Mutex
}

func (b *binary) Name() string {
	return b.name
}

func (b *binary) Init(config interface{}, mgr api.SidecarManager) error {
	// 初始化插件配置
	if cfg, ok := config.(*v1alpha1.Binary); ok {
		b.config = *cfg
		return nil
	}
	return errors.New("invalid config type")
}

func (b *binary) Start(ctx context.Context, errCh chan<- error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cmd = exec.CommandContext(ctx, b.config.Path, b.config.Args...)
	b.cmd.Env = append(os.Environ(), b.config.Env...)
	b.cmd.Stdout = os.Stdout
	b.cmd.Stderr = os.Stderr
	if err := b.cmd.Start(); err != nil {
		errCh <- err
		return
	}

	go func() {
		errCh <- b.cmd.Wait()
	}()
}

func (b *binary) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.cmd != nil && b.cmd.Process != nil {
		return b.cmd.Process.Kill()
	}
	return nil
}

func (b *binary) Version() string {
	return b.config.Version
}

func (b *binary) Status() (*api.PluginStatus, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.status == nil {
		return nil, errors.New("status not available")
	}
	return b.status, nil
}

func (b *binary) GetConfigType() interface{} {
	return &binary{}
}

func (b *binary) updateStatus() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.status = &api.PluginStatus{
		Name:        b.Name(),
		Version:     b.config.Version,
		Running:     b.cmd != nil && b.cmd.Process != nil,
		LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		Health:      "Healthy",
		Infos:       []string{fmt.Sprintf("Binary path: %s", b.config.Path)},
	}
}
