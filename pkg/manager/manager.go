package manager

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type SidecarManager struct{
	ctrl.Manager
	DBManager
}

type DBManager struct{

}

func NewManager () (*SidecarManager,error){
	cfg, err := config.GetConfig()
	if err != nil {
		return nil,fmt.Errorf("unable to get kubeconfig: %w", err)
	}

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		return nil,fmt.Errorf("unable to create manager: %w", err)
	}
	return &SidecarManager{Manager:mgr},nil
}