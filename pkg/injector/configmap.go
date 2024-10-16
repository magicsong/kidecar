package injector

import (
	"context"
	"fmt"

	"github.com/magicsong/kidecar/api/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)


func buildConfigYaml(SidecarConfig *v1alpha1.SidecarConfig) (string, error) {
	// 将结构体转换为 YAML
	yamlData, err := yaml.Marshal(SidecarConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal SidecarConfig to YAML: %v", err)
	}
	return string(yamlData), nil
}

func createConfigmap(ctx context.Context, ctrlclient client.Client, namespace string, SidecarConfig *v1alpha1.SidecarConfig) error {
	configYaml, err := buildConfigYaml(SidecarConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal SidecarConfig to YAML: %v", err)
	}
	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KidecarConfigmapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"config.yaml": configYaml,
		},
	}
	controllerutil.SetOwnerReference(SidecarConfig, configmap, ctrlclient.Scheme())
	if err := ctrlclient.Create(ctx, configmap); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		logf.FromContext(ctx).Error(err, "failed to create configmap")
		return err
	}
	return nil
}