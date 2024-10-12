package injector

import (
	"context"
	"fmt"

	"github.com/magicsong/kidecar/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	InjectAnnotationKey = "sidecarconfig.kidecar.io/inject"
)

func GetSidecarConfigOfPod(ctx context.Context, pod *corev1.Pod, ctrlclient client.Client) (*v1alpha1.SidecarConfig, error) {
	sidecarconfigList := &v1alpha1.SidecarConfigList{}
	if err := ctrlclient.List(ctx, sidecarconfigList); err != nil {
		return nil, fmt.Errorf("failed to list SidecarConfig: %v", err)
	}
	for _, sidecarconfig := range sidecarconfigList.Items {
		if matched, err := Match(pod, &sidecarconfig.Spec, ctrlclient); err != nil {
			return nil, fmt.Errorf("failed to match SidecarConfig: %v", err)
		} else {
			if matched {
				return &sidecarconfig, nil
			}
		}
	}
	return nil, nil
}

func Match(pod *corev1.Pod, sidecarconfig *v1alpha1.SidecarConfigSpec, ctrlclient client.Client) (bool, error) {
	if sidecarconfig.NamespaceSelector != nil {
		if matched, err := MatchNamespace(pod.Namespace, sidecarconfig.NamespaceSelector, ctrlclient); err != nil {
			return false, err
		} else {
			if !matched {
				return false, nil
			}
		}
	}
	if sidecarconfig.Selector != nil {
		if matched, err := MatchLabelSelector(sidecarconfig.Selector, pod.Labels); err != nil {
			return false, err
		} else {
			if !matched {
				return false, nil
			}
		}
	}
	return true, nil
}

func MatchLabelSelector(selector *metav1.LabelSelector, podLabels map[string]string) (bool, error) {
	if selector == nil {
		return true, nil
	}
	s, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return false, err
	}
	return s.Matches(labels.Set(podLabels)), nil
}

func MatchNamespace(namespace string, selector *metav1.LabelSelector, ctrlclient client.Client) (bool, error) {
	if selector == nil {
		return true, nil
	}
	s, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return false, err
	}
	nsObj := &corev1.Namespace{}
	if err := ctrlclient.Get(context.Background(), client.ObjectKey{Name: namespace}, nsObj); err != nil {
		return false, fmt.Errorf("failed to get namespace %s: %v", namespace, err)
	}
	return s.Matches(labels.Set(nsObj.Labels)), nil
}

func InjectPod(ctx context.Context, pod *corev1.Pod, ctrlclient client.Client) error {
	log := logf.FromContext(ctx)
	config, err := GetSidecarConfigOfPod(ctx, pod, ctrlclient)
	if err != nil {
		return err
	}
	if config == nil {
		log.Info("no SidecarConfig matched, skip injecting")
		return nil
	}
	injectServiceAccount(pod, config.Spec.ServiceAccountName, config.Spec.ForceInjectServiceAccount != nil && *config.Spec.ForceInjectServiceAccount)
	addContainers(pod, config.Spec.Containers)
	addInitContainers(pod, config.Spec.InitContainers)
	addVolumes(pod, config.Spec.Volumes)
	addVolumeMounts(pod, config.Spec.VolumeMounts)
	shareProcessNamespace(pod, config.Spec.ShareProcessNamespace != nil && *config.Spec.ShareProcessNamespace)
	addAnnotations(pod, config.Annotations)

	// update sidecar config status
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		originConfig := &v1alpha1.SidecarConfig{}
		if err := ctrlclient.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: config.Namespace}, originConfig); err != nil {
			return err
		}
		originConfig.Status.MatchedPods++
		return ctrlclient.Status().Update(ctx, originConfig)
	})
	if err != nil {
		// will continue to inject even if failed to update status
		log.Error(err, "failed to update SidecarConfig status after retrying")
	}
	return nil
}

func injectServiceAccount(pod *corev1.Pod, serviceAccountName string, force bool) {
	// be sure to inject the serviceAccountName before adding any volumeMounts, because we must prune out any existing
	// volumeMounts that were added to support the default service account. Because this removal is by index, we splice
	// them out before appending new volumes at the end.
	if serviceAccountName != "" && (pod.Spec.ServiceAccountName == "default" || pod.Spec.ServiceAccountName == "") {
		pod.Spec.ServiceAccountName = serviceAccountName
	}
	if force {
		pod.Spec.ServiceAccountName = serviceAccountName
	}
}
func addContainers(pod *corev1.Pod, containers []corev1.Container) {
	if len(containers) == 0 {
		return
	}
	pod.Spec.Containers = append(containers, pod.Spec.Containers...)
}

func addInitContainers(pod *corev1.Pod, initContainers []corev1.Container) {
	if len(initContainers) == 0 {
		return
	}
	pod.Spec.InitContainers = append(initContainers, pod.Spec.InitContainers...)
}

func addVolumes(pod *corev1.Pod, volumes []corev1.Volume) {
	if len(volumes) == 0 {
		return
	}
	pod.Spec.Volumes = append(volumes, pod.Spec.Volumes...)
}

func shareProcessNamespace(pod *corev1.Pod, shareProcessNamespace bool) {
	if shareProcessNamespace {
		pod.Spec.ShareProcessNamespace = &shareProcessNamespace
	}

}

func addVolumeMounts(pod *corev1.Pod, volumeMounts []corev1.VolumeMount) {
	for _, container := range pod.Spec.Containers {
		container.VolumeMounts = append(volumeMounts, container.VolumeMounts...)
	}
	for _, container := range pod.Spec.InitContainers {
		container.VolumeMounts = append(volumeMounts, container.VolumeMounts...)
	}
}

func addAnnotations(pod *corev1.Pod, annotations map[string]string) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	for k, v := range annotations {
		pod.Annotations[k] = v
	}
	pod.Annotations[InjectAnnotationKey] = "true"
}
