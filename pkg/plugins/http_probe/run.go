package httpprobe

import (
	"context"

	"github.com/magicsong/okg-sidecar/api"
)

type httpProber struct{}

// Init implements api.Plugin.
func (h *httpProber) Init(config map[string]interface{}) error {
	panic("unimplemented")
}

// Name implements api.Plugin.
func (h *httpProber) Name() string {
	panic("unimplemented")
}

// Start implements api.Plugin.
func (h *httpProber) Start(ctx context.Context) error {
	panic("unimplemented")
}

// Status implements api.Plugin.
func (h *httpProber) Status() (*api.PluginStatus, error) {
	panic("unimplemented")
}

// Stop implements api.Plugin.
func (h *httpProber) Stop(ctx context.Context) error {
	panic("unimplemented")
}

// Version implements api.Plugin.
func (h *httpProber) Version() string {
	panic("unimplemented")
}

func NewPlugin() api.Plugin {
	return &httpProber{}
}
