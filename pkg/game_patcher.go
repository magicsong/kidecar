package pkg

import (
	"context"
	"fmt"

	kruisegameclientset "github.com/openkruise/kruise-game/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrlLog "sigs.k8s.io/controller-runtime/pkg/log"
)

type GamePatcher interface {
	// PatchGameServerStatus patches the game server status
	PatchGameServerStatus(ctx context.Context, name, namespace string, status string) error
}

type okgGamePatcher struct {
	KubeClientset *kubernetes.Clientset
	*kruisegameclientset.Clientset
}

func NewGamePatcher(kubeClientset *kubernetes.Clientset, okgClientset *kruisegameclientset.Clientset) GamePatcher {
	return &okgGamePatcher{
		KubeClientset: kubeClientset,
		Clientset:     okgClientset,
	}
}

func (o *okgGamePatcher) PatchGameServerStatus(ctx context.Context, name, namespace string, status string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// 获取指定名称的gameserverset
		gameServerSet, err := o.GameV1alpha1().GameServerSets(namespace).Get(ctx, name, metav1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			ctrlLog.Log.Error(err, "Get GameServerSet err")
			return err
		}
		// 修改gameserverset的副本数
		gameServerSet, err = o.GameV1alpha1().GameServerSets(namespace).Update(ctx, gameServerSet, metav1.UpdateOptions{})
		if err != nil {
			ctrlLog.Log.Error(err, "Update GameServerSet err")
			return err
		}
		fmt.Println("GameServerSet ready, gameServerSet.Spec.Replicas: ", *gameServerSet.Spec.Replicas)
		return nil
	})
}
