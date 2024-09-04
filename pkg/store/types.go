package store

import "github.com/magicsong/okg-sidecar/api"

type Storage interface {
	IsInitialized() bool
	SetupWithManager(mgr api.SidecarManager) error
	Store(data interface{}, config interface{}) error
}
