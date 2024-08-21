package e2e

import (
	"context"

	"github.com/magicsong/okg-sidecar/api"
)

var _ api.Plugin = &SamplePlugin{}

type SamplePlugin struct{}

// Init implements api.Plugin.
func (s *SamplePlugin) Init(config map[string]interface{}) error {
	panic("unimplemented")
}

// Name implements api.Plugin.
func (s *SamplePlugin) Name() string {
	panic("unimplemented")
}

// Start implements api.Plugin.
func (s *SamplePlugin) Start(ctx context.Context) error {
	panic("unimplemented")
}

// Status implements api.Plugin.
func (s *SamplePlugin) Status() (*api.PluginStatus, error) {
	panic("unimplemented")
}

// Stop implements api.Plugin.
func (s *SamplePlugin) Stop(ctx context.Context) error {
	panic("unimplemented")
}

// Version implements api.Plugin.
func (s *SamplePlugin) Version() string {
	panic("unimplemented")
}
