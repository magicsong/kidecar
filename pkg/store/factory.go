package store

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
)

type StorageFactory interface {
	GetStorage(storageType StorageType) (Storage, error)
}

type defaultStorageFactory struct {
	storageMap map[StorageType]Storage
	manager    ctrl.Manager
}

func NewStorageFactory() StorageFactory {
	f := &defaultStorageFactory{
		storageMap: make(map[StorageType]Storage),
	}
	f.storageMap[StorageTypePodAnnotation] = &PodAnnotationStore{}
	return f
}

func (f *defaultStorageFactory) GetStorage(storageType StorageType) (Storage, error) {
	s := f.storageMap[storageType]
	if s == nil {
		return nil, fmt.Errorf("storage type %s not found", storageType)
	}
	if !s.IsInitialized() {
		if err := s.SetupWithManager(f.manager); err != nil {
			return nil, fmt.Errorf("failed to setup storage: %w", err)
		}
	}
	return s, nil
}
