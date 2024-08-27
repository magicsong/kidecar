package store

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ Storage = &PodAnnotationStore{}

type Storage interface {
	Store(data interface{}, config interface{}) error
	SetupWithManager(mgr ctrl.Manager) error
	IsInitialized() bool
}

type PodAnnotationStore struct {
	client.Client
	// maybe we need dynamic client

}

// IsInitialized implements Storage.
func (s *PodAnnotationStore) IsInitialized() bool {
	return s.Client != nil
}

// SetupWithManager implements Storage.
func (s *PodAnnotationStore) SetupWithManager(mgr manager.Manager) error {
	s.Client = mgr.GetClient()
	return nil
}

func (s *PodAnnotationStore) Store(data interface{}, config interface{}) error {
	if config == nil {
		return fmt.Errorf("invalid config")
	}
	myConfig, ok := config.(*PodAnnotationConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}
	nsName, err := getCurrentPodNamespaceAndName()
	if err != nil {
		return err
	}
	// get pod
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pod := &corev1.Pod{}
		if err := s.Get(context.Background(), *nsName, pod); err != nil {
			return fmt.Errorf("failed to get pod: %w", err)
		}
		newPod := pod.DeepCopy()
		newPod.Annotations[myConfig.AnnotationKey] = fmt.Sprintf("%v", data)
		return s.Client.Patch(context.Background(), newPod, client.MergeFrom(pod))
	})
	if err != nil {
		return fmt.Errorf("failed to patch pod: %w", err)
	}
	return nil
}

func (s *PodAnnotationStore) Name() string {
	return "PodAnnotation"
}

func getCurrentPodNamespaceAndName() (*types.NamespacedName, error) {
	ns := os.Getenv("POD_NAMESPACE")
	name := os.Getenv("POD_NAME")
	if ns == "" || name == "" {
		return nil, fmt.Errorf("failed to get current pod namespace and name")
	}
	return &types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, nil
}
