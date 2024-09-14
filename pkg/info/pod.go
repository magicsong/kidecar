package info

import (
	"context"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// GetCurrentPod return pod the sidecar running
var cacheExpiration time.Duration = 1 * time.Minute
var cache map[string]*corev1.Pod = make(map[string]*corev1.Pod)

func GetCurrentPod() (*corev1.Pod, error) {
	nsname, err := GetCurrentPodNamespaceAndName()
	if err != nil {
		return nil, err
	}
	// Check if the pod is already cached
	if pod, ok := cache[nsname.String()]; ok {
		return pod, nil
	}

	// Fetch the pod from the Kubernetes API
	pod, err := globalKubeInterface.CoreV1().Pods(nsname.Namespace).Get(context.TODO(), nsname.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Cache the pod for future use
	cache[nsname.String()] = pod

	// Set a timer to expire the cache entry after the specified duration
	time.AfterFunc(cacheExpiration, func() {
		delete(cache, nsname.String())
	})

	return pod, nil
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

func GetCurrentPodInfo() (string, error) {
	ns := os.Getenv("POD_NAMESPACE")
	name := os.Getenv("POD_NAME")
	if ns == "" || name == "" {
		return "", fmt.Errorf("failed to get current pod namespace and name")
	}

	return fmt.Sprintf("%s-%s", ns, name), nil

}

var globalKubeInterface kubernetes.Interface

func SetGlobalKubeInterface(k8sClient kubernetes.Interface) {
	globalKubeInterface = k8sClient
}
