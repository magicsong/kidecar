package store

import "github.com/magicsong/kidecar/api"

type Storage interface {
	IsInitialized() bool
	SetupWithManager(mgr api.SidecarManager) error
	Store(data string, config interface{}) error
}
