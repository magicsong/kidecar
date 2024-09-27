package info

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetConfigmap(ctx context.Context, name, nemaspace string) (*corev1.ConfigMap, error) {
	cm, err := globalKubeInterface.CoreV1().ConfigMaps(nemaspace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func UpdateConfigmap(ctx context.Context, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	cm, err := globalKubeInterface.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return cm, nil
}
