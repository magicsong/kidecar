package webhook

/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/magicsong/kidecar/pkg/injector"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=ignore,groups="",resources=pods,verbs=create,versions=v1,name=mpod.vkeengine.com,sideEffects=None,admissionReviewVersions=v1

// podInjector annotates Pods
type PodInjector struct {
	Client client.Client
}

func (a *PodInjector) Default(ctx context.Context, obj runtime.Object) error {
	log := logf.FromContext(ctx)
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod but got a %T", obj)
	}
	log.Info("before inject", "podSpec", pod.Spec)
	if err := injector.InjectPod(ctx, pod, a.Client); err != nil {
		return fmt.Errorf("failed to inject pod: %w", err)
	}
	log.Info("after inject", "podSpec", pod.Spec)
	return nil
}

func (p *PodInjector) SetupWebhookWithManager(manager ctrl.Manager) error {
	p.Client = manager.GetClient()
	return ctrl.NewWebhookManagedBy(manager).For(&corev1.Pod{}).WithDefaulter(p).Complete()
}
