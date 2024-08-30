package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ Storage = &PodAnnotationStore{}

type Storage interface {
	Store(data interface{}, config interface{}) error
	SetupWithManager(mgr ctrl.Manager) error
	IsInitialized() bool
}

type PodAnnotationStore struct {
	kubeClient kubernetes.Interface
	// maybe we need dynamic client

}

// IsInitialized implements Storage.
func (s *PodAnnotationStore) IsInitialized() bool {
	return s.kubeClient != nil
}

// SetupWithManager implements Storage.
func (s *PodAnnotationStore) SetupWithManager(mgr manager.Manager) error {
	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to create kube client: %w", err)
	}
	s.kubeClient = clientset
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
		patchData := map[string]interface{}{
			"metadata": map[string]interface{}{
				"annotations": map[string]string{
					myConfig.AnnotationKey: fmt.Sprintf("%v", data),
				},
			},
		}
		patchBytes, _ := json.Marshal(patchData)
		_, err = s.kubeClient.CoreV1().Pods(nsName.Namespace).Patch(context.Background(), nsName.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			return err
		}
		return nil
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
