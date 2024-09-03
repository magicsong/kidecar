package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/magicsong/okg-sidecar/pkg/info"
	"gomodules.xyz/jsonpatch/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ Storage = &crd{}

type crd struct {
	log     logr.Logger
	dynamic dynamic.Interface
	kubernetes.Interface
}

// IsInitialized implements Storage.
func (c *crd) IsInitialized() bool {
	return c.Interface != nil
}

// SetupWithManager implements Storage.
func (c *crd) SetupWithManager(mgr manager.Manager) error {
	dynClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	c.dynamic = dynClient
	c.Interface = kubernetes.NewForConfigOrDie(mgr.GetConfig())
	c.log = mgr.GetLogger().WithName("crd_store")
	return nil
}

// Store implements Storage.
func (c *crd) Store(data interface{}, config interface{}) error {
	myconfig, ok := config.(*CRDConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}
	if err := myconfig.IsValid(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	pod, err := info.GetCurrentPod(c)
	if err != nil {
		return fmt.Errorf("failed to get current pod: %w", err)
	}
	if err := myconfig.ParseTemplate(&pod.Spec.Containers[0]); err != nil {
		return fmt.Errorf("failed to parse template in crd config: %w", err)
	}
	gvr := schema.GroupVersionResource{Group: myconfig.Group, Version: myconfig.Version, Resource: myconfig.Resource}
	patch := []jsonpatch.JsonPatchOperation{{
		Operation: "replace",
		Path:      myconfig.JsonPath,
		Value:     data},
	}
	patchBytes, _ := json.Marshal(patch)
	c.log.Info("patch crd", "crd", myconfig, "patch", string(patchBytes), "gvr", gvr)
	_, err = c.dynamic.Resource(gvr).Namespace(myconfig.Namespace).Patch(context.TODO(), myconfig.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch crd: %w", err)
	}
	c.log.Info("store data", "data", data, "crd", myconfig.Name)
	return nil
}
