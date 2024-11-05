package collector

import (
	"context"
	"fmt"
	"time"

	serverlessv1alpha1 "github.com/magicsong/kidecar/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type podInfoCollector struct {
	client client.Client
}

// StartPodInfoCollector starts a new pod info collector and registers it with the manager
func StartPodInfoCollector(mgr manager.Manager) error {
	p := &podInfoCollector{
		client: mgr.GetClient(),
	}
	return mgr.Add(p)
}

func (p *podInfoCollector) syncPodInfoToSidecarSet() {
	// 获取 SidecarConfig 并更新其 Status 状态
	var sidecarConfigList serverlessv1alpha1.SidecarConfigList
	err := p.client.List(context.TODO(), &sidecarConfigList)
	if err != nil {
		logf.Log.Error(err, "failed to list sidecarconfig")
		return
	}
	for _, sidecarConfig := range sidecarConfigList.Items {
		// 更新 SidecarConfig 的 Status 状态
		logf.Log.Info("collecting pod info", "sidecarConfig", sidecarConfig.Name)
		defer func() {
			if r := recover(); r != nil {
				logf.Log.Error(fmt.Errorf("panic in collecting pod info: %v", r), "recovering")
			}
		}()
		err = p.collectPodAndUpdateStatus(&sidecarConfig)
		if err != nil {
			logf.Log.Info("failed to collect pod info", "sidecarConfig", sidecarConfig.Name, "error", err)
			continue
		}
	}
}

func (p *podInfoCollector) collectPodAndUpdateStatus(sidecarConfig *serverlessv1alpha1.SidecarConfig) error {
	podList := &corev1.PodList{}
	s, err := metav1.LabelSelectorAsSelector(sidecarConfig.Spec.Injection.Selector)
	if err != nil {
		return fmt.Errorf("failed to convert label selector: %v", err)
	}
	opts := make([]client.ListOption, 0)
	if sidecarConfig.Spec.Injection.Namespace != "" {
		opts = append(opts, client.InNamespace(sidecarConfig.Spec.Injection.Namespace))
	}
	if s != nil {
		opts = append(opts, client.MatchingLabelsSelector{Selector: s})
	}
	if err = p.client.List(context.TODO(), podList, opts...); err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}
	// caculate UPDATEDPODS   READYPODS
	updatedPods, readyPods := 0, 0
	for _, pod := range podList.Items {
		updatedPods++
		if pod.Status.Phase == corev1.PodRunning {
			readyPods++
		}
	}
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := p.client.Get(context.TODO(), client.ObjectKeyFromObject(sidecarConfig), sidecarConfig); err != nil {
			return err
		}
		sidecarConfig.Status.UpdatedPods = int32(updatedPods)
		sidecarConfig.Status.ReadyPods = int32(readyPods)
		return p.client.Status().Update(context.Background(), sidecarConfig)
	})
	if retryErr != nil {
		return fmt.Errorf("failed to update SidecarConfig status: %w", retryErr)
	}
	return nil
}

func (p *podInfoCollector) Start(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			p.syncPodInfoToSidecarSet()
		}
	}
}
