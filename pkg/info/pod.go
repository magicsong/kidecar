package info

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// GetCurrentPod return pod the sidecar running
func GetCurrentPod(k8sClient kubernetes.Interface) (*corev1.Pod, error) {
	nsname, err := GetCurrentPodNamespaceAndName()
	if err != nil {
		return nil, err
	}
	return k8sClient.CoreV1().Pods(nsname.Namespace).Get(context.TODO(), nsname.Name, metav1.GetOptions{})
}

func GetCurrentPodNamespaceAndName() (*types.NamespacedName, error) {
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
