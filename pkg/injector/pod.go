package injector

import (
	"context"
	"fmt"
	"os"

	"github.com/magicsong/kidecar/api/v1alpha1"
	"github.com/magicsong/kidecar/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	InjectAnnotationKey  = "sidecarconfig.kidecar.io/inject"
	ConfigmapHashKey     = "sidecarconfig.kidecar.io/hash"
	KidecarConfigmapName = "kidecar-config"
	HotUpdateVolume      = "share-data"
)

const (
	EnvKidecarImage     string = "KIDECAR_IMAGE"
	DefaultKidecarImage string = "113745426946.dkr.ecr.us-east-1.amazonaws.com/xuetaotest/kidecar:v5"

	serviceAccountTokenMountPath = "/var/run/secrets/kubernetes.io/serviceaccount"
)

func GetSidecarConfigOfPod(ctx context.Context, pod *corev1.Pod, ctrlclient client.Client) (*v1alpha1.SidecarConfig, error) {
	sidecarconfigList := &v1alpha1.SidecarConfigList{}
	if err := ctrlclient.List(ctx, sidecarconfigList); err != nil {
		return nil, fmt.Errorf("failed to list SidecarConfig: %v", err)
	}
	for index, sidecarconfig := range sidecarconfigList.Items {
		if matched, err := Match(pod, &sidecarconfig.Spec, ctrlclient); err != nil {
			return nil, fmt.Errorf("failed to match SidecarConfig: %v", err)
		} else {
			if matched {
				return &sidecarconfigList.Items[index], nil
			}
		}
	}
	return nil, nil
}

func Match(pod *corev1.Pod, sidecarconfig *v1alpha1.SidecarConfigSpec, ctrlclient client.Client) (bool, error) {
	if sidecarconfig.Injection.NamespaceSelector != nil {
		if matched, err := MatchNamespace(pod.Namespace, sidecarconfig.Injection.NamespaceSelector, ctrlclient); err != nil {
			return false, err
		} else {
			if !matched {
				return false, nil
			}
		}
	}
	if sidecarconfig.Injection.Selector != nil {
		if matched, err := MatchLabelSelector(sidecarconfig.Injection.Selector, pod.Labels); err != nil {
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
		log.Info("no SidecarConfig.Injection matched, skip injecting")
		return nil
	}
	// check inject annotation
	if _, ok := pod.Annotations[InjectAnnotationKey]; ok {
		log.Info("pod already injected, skip injecting")
		return nil
	}
	if err := injectServiceAccount(pod, config.Spec.Injection.ServiceAccountName, config.Spec.Injection.ForceInjectServiceAccount != nil && *config.Spec.Injection.ForceInjectServiceAccount, ctrlclient); err != nil {
		log.Error(err, "failed to inject service account")
		return err
	}
	if config.Spec.Injection.InjectKidecar {
		addKidecarContainer(pod, config)
		log.Info("inject kidecar container")
		defer log.Info("inject kidecar container DONE")
		if err := EnsureConfigmap(ctx, ctrlclient, pod.Namespace, config); err != nil {
			return err
		}
	} else {
		log.Info("skip injecting kidecar container, inject user defined sidecar")
		addContainers(pod, config.Spec.Injection.Containers)
		addInitContainers(pod, config.Spec.Injection.InitContainers)
	}
	addVolumes(pod, config.Spec.Injection.Volumes)
	addVolumeMounts(pod, config.Spec.Injection.VolumeMounts)
	shareProcessNamespace(pod, config.Spec.Injection.ShareProcessNamespace != nil && *config.Spec.Injection.ShareProcessNamespace)
	addAnnotations(pod, config.Annotations)

	// update sidecar config status
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		originConfig := &v1alpha1.SidecarConfig{}
		if err := ctrlclient.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: config.Namespace}, originConfig); err != nil {
			return err
		}
		originConfig.Status.MatchedPods = config.Status.MatchedPods + 1
		return ctrlclient.Status().Update(ctx, originConfig)
	})
	if err != nil {
		// will continue to inject even if failed to update status
		log.Error(err, "failed to update SidecarConfig.Injection status after retrying")
	}
	return nil
}

func injectServiceAccount(pod *corev1.Pod, serviceAccountName string, force bool, client client.Client) error {
	// check service account exist
	if serviceAccountName != "" {
		sa := &corev1.ServiceAccount{}
		if err := client.Get(context.Background(), types.NamespacedName{Name: serviceAccountName, Namespace: pod.Namespace}, sa); err != nil {
			return fmt.Errorf("failed to get service account %s: %v", serviceAccountName, err)
		}
	}
	// be sure to inject the serviceAccountName before adding any volumeMounts, because we must prune out any existing
	// volumeMounts that were added to support the default service account. Because this removal is by index, we splice
	// them out before appending new volumes at the end.
	if serviceAccountName != "" && (pod.Spec.ServiceAccountName == "default" || pod.Spec.ServiceAccountName == "") {
		pod.Spec.ServiceAccountName = serviceAccountName
	}
	if force {
		pod.Spec.ServiceAccountName = serviceAccountName
	}
	for i, container := range pod.Spec.Containers {
		// remove system injected volumeMounts
		for j, volumeMount := range container.VolumeMounts {
			if volumeMount.MountPath == serviceAccountTokenMountPath {
				if len(container.VolumeMounts) == 1 {
					pod.Spec.Containers[i].VolumeMounts = []corev1.VolumeMount{}
				} else {
					pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts[:j], pod.Spec.Containers[i].VolumeMounts[j+1:]...)
				}
				break
			}
		}
	}
	return nil
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

func getKidecarImage() string {
	image := os.Getenv(EnvKidecarImage)
	if image == "" {
		return DefaultKidecarImage
	}
	return image
}

func addKidecarContainer(pod *corev1.Pod, SidecarConfig *v1alpha1.SidecarConfig) {
	kContainer := corev1.Container{
		Name:  "kidecar",
		Image: getKidecarImage(),
		Env: []corev1.EnvVar{
			//PodName
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			//PodNamespace
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			// PROCESS_NAME
			{
				Name:  "PROCESS_NAME",
				Value: "nginx: master process nginx",
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "hot-update-port",
				ContainerPort: 5000,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      KidecarConfigmapName,
				MountPath: "/opt/kidecar",
			},
		},
	}
	if SidecarConfig.Spec.Injection.UseKubeNativeSidecar {
		kContainer.RestartPolicy = utils.ConvertAnyToPtr(corev1.ContainerRestartPolicyAlways)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, kContainer)
	} else {
		pod.Spec.Containers = append(pod.Spec.Containers, kContainer)
	}

	// add volumes
	if pod.Spec.Volumes == nil {
		pod.Spec.Volumes = []corev1.Volume{}
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: KidecarConfigmapName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: KidecarConfigmapName,
				},
			},
		},
	})
}
