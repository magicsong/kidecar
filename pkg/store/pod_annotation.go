package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/magicsong/okg-sidecar/pkg/info"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ Storage = &podAnnotationStore{}

type Storage interface {
	Store(data interface{}, config interface{}) error
	SetupWithManager(mgr ctrl.Manager) error
	IsInitialized() bool
}

type podAnnotationStore struct {
	kubeClient kubernetes.Interface
	// maybe we need dynamic client
	log logr.Logger
}

// IsInitialized implements Storage.
func (s *podAnnotationStore) IsInitialized() bool {
	return s.kubeClient != nil
}

// SetupWithManager implements Storage.
func (s *podAnnotationStore) SetupWithManager(mgr manager.Manager) error {
	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to create kube client: %w", err)
	}
	s.kubeClient = clientset
	s.log = mgr.GetLogger().WithName("pod_annotation_store")
	return nil
}

func (s *podAnnotationStore) Store(data interface{}, config interface{}) error {
	if config == nil {
		return fmt.Errorf("invalid config")
	}
	myConfig, ok := config.(*PodAnnotationConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}
	nsName, err := info.GetCurrentPodNamespaceAndName()
	if err != nil {
		return err
	}
	s.log.Info("store data", "data", data, "pod", nsName)
	defer s.log.Info("store data done", "data", data, "pod", nsName)
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

func (s *podAnnotationStore) Name() string {
	return "PodAnnotation"
}
