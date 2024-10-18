package injector

import (
	"context"
	"fmt"

	"github.com/magicsong/kidecar/api/v1alpha1"
	"github.com/magicsong/kidecar/pkg/utils"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func buildConfigYaml(kidecarConfig *v1alpha1.KidecarConfig) (string, error) {
	// 将结构体转换为 YAML
	yamlData, err := yaml.Marshal(kidecarConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal kidecarConfig to YAML: %v", err)
	}
	return string(yamlData), nil
}

func EnsureConfigmap(ctx context.Context, ctrlclient client.Client, namespace string, SidecarConfig *v1alpha1.SidecarConfig) error {
	configYaml, err := buildConfigYaml(&SidecarConfig.Spec.Kidecar)
	if err != nil {
		return fmt.Errorf("failed to marshal SidecarConfig to YAML: %v", err)
	}
	configmap := &corev1.ConfigMap{}
	if err := ctrlclient.Get(ctx, client.ObjectKey{Name: KidecarConfigmapName, Namespace: namespace}, configmap); err != nil {
		if apierrors.IsNotFound(err) {
			return createConfigmap(ctx, ctrlclient, namespace, &configYaml, SidecarConfig)
		}
		logf.FromContext(ctx).Error(err, "failed to get configmap")
		return err
	}
	if isConfigChanged(configYaml, configmap) {
		return updateConfigmap(ctx, ctrlclient, namespace, &configYaml, SidecarConfig)
	}
	return nil
}

func updateConfigmap(ctx context.Context, ctrlclient client.Client, namespace string, configYaml *string, SidecarConfig *v1alpha1.SidecarConfig) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		configmap := &corev1.ConfigMap{}
		if err := ctrlclient.Get(ctx, client.ObjectKey{Name: KidecarConfigmapName, Namespace: namespace}, configmap); err != nil {
			logf.FromContext(ctx).Error(err, "failed to get configmap")
			return err
		}
		configmap.Data["config.yaml"] = *configYaml
		hash := utils.Hash(*configYaml)
		configmap.Annotations[ConfigmapHashKey] = hash
		if err := ctrlclient.Update(ctx, configmap); err != nil {
			logf.FromContext(ctx).Error(err, "failed to update configmap")
			return err
		}
		return nil
	})
}

func createConfigmap(ctx context.Context, ctrlclient client.Client, namespace string, configYaml *string, SidecarConfig *v1alpha1.SidecarConfig) error {
	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KidecarConfigmapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"config.yaml": *configYaml,
		},
	}
	controllerutil.SetOwnerReference(SidecarConfig, configmap, ctrlclient.Scheme())
	hash := utils.Hash(*configYaml)
	configmap.Annotations = map[string]string{
		ConfigmapHashKey: hash,
	}
	if err := ctrlclient.Create(ctx, configmap); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		logf.FromContext(ctx).Error(err, "failed to create configmap")
		return err
	}
	return nil
}

func isConfigChanged(configmapYaml string, configmap *corev1.ConfigMap) bool {
	hash := utils.Hash(configmapYaml)
	return configmap.Annotations[ConfigmapHashKey] != hash
}
