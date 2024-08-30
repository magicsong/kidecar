package manager

import (
	"fmt"

	"github.com/magicsong/okg-sidecar/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type sidecarManager struct {
	ctrl.Manager
	api.DBManager
}

func NewManager() (api.SidecarManager, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to get kubeconfig: %w", err)
	}

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		return nil, fmt.Errorf("unable to create manager: %w", err)
	}
	return sidecarManager{Manager: mgr}, nil
}
