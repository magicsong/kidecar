package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/magicsong/okg-sidecar/api"
	"github.com/magicsong/okg-sidecar/pkg/info"
	"gomodules.xyz/jsonpatch/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

var _ Storage = &inKube{}

type inKube struct {
	log     logr.Logger
	dynamic dynamic.Interface
	kubernetes.Interface
}

// IsInitialized implements Storage.
func (c *inKube) IsInitialized() bool {
	return c.Interface != nil
}

// SetupWithManager implements Storage.
func (c *inKube) SetupWithManager(mgr api.SidecarManager) error {
	dynClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}
	c.log = mgr.GetLogger().WithName("in_kube")
	c.dynamic = dynClient
	c.Interface = mgr
	return nil
}

func (c *inKube) storeInCurrentPod(data interface{}, config *InKubeConfig) error {
	currentPod, err := info.GetCurrentPod()
	if err != nil {
		return fmt.Errorf("failed to get current pod: %w", err)
	}
	c.log.Info("store data in current pod", "data", data, "name", currentPod.Name)
	defer c.log.Info("store data done", "data", data, "pod", currentPod.Name)
	// get pod
	metadata := make(map[string]interface{})
	patchData := map[string]interface{}{
		"metadata": map[string]interface{}{},
	}
	if config.AnnotationKey != nil {
		metadata["annotations"] = map[string]interface{}{*config.AnnotationKey: data}
	}
	if config.LabelKey != nil {
		metadata["labels"] = map[string]interface{}{*config.LabelKey: data}
	}
	patchData["metadata"] = metadata
	patchBytes, _ := json.Marshal(patchData)
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err = c.CoreV1().Pods(currentPod.Namespace).Patch(context.Background(), currentPod.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to patch pod after mant retries: %w", err)
	}
	return nil
}

// Store implements Storage.
func (c *inKube) Store(data interface{}, config interface{}) error {
	myconfig, ok := config.(*InKubeConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}
	c.log.Info("store data", "data", data, "inKube", myconfig)
	defer c.log.Info("store data done", "data", data)
	if err := myconfig.IsValid(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if myconfig.Target == nil {
		return c.storeInCurrentPod(data, myconfig)
	}
	return c.storeInOtherObject(data, myconfig)
}

func (c *inKube) storeInOtherObject(data interface{}, myconfig *InKubeConfig) error {
	if myconfig.Target == nil || len(myconfig.MarkerPolices) < 1 {
		return fmt.Errorf("invalid target or markerPolices")
	}
	gvr := myconfig.Target.ToGvr()
	c.log.Info("store data in other object", "data", data, "inKube", myconfig, "gvr", gvr)
	patch := generatePatch(data, myconfig)
	patchBytes, _ := json.Marshal(patch)
	c.log.Info("patch inKube", "inKube", myconfig, "patch", string(patchBytes), "gvr", gvr)
	_, err := c.dynamic.Resource(gvr).Namespace(myconfig.Target.Namespace).Patch(context.TODO(), myconfig.Target.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch inKube: %w", err)
	}
	return nil
}

func generatePatch(data interface{}, myconfig *InKubeConfig) []jsonpatch.JsonPatchOperation {
	patch := []jsonpatch.JsonPatchOperation{}
	if myconfig.AnnotationKey != nil {
		patch = append(patch, jsonpatch.NewOperation("replace", "/metadata/annotations/"+*myconfig.AnnotationKey, data))
	}
	if myconfig.LabelKey != nil {
		patch = append(patch, jsonpatch.NewOperation("replace", "/metadata/labels/"+*myconfig.LabelKey, data))
	}
	if len(myconfig.MarkerPolices) > 0 {
		for _, policy := range myconfig.MarkerPolices {
			if policy.State == fmt.Sprintf("%v", data) {
				if len(policy.Labels) > 0 {
					for key, value := range policy.Labels {
						patch = append(patch, jsonpatch.NewOperation("replace", "/metadata/labels/"+key, value))
					}
				}
				if len(policy.Annotations) > 0 {
					for key, value := range policy.Annotations {
						patch = append(patch, jsonpatch.NewOperation("replace", "/metadata/annotations/"+key, value))
					}
				}
				break
			}
		}
	}
	return patch
}
