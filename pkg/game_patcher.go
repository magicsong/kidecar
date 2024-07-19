package pkg

import (
	"context"
	"github.com/openkruise/kruise-game/apis/v1alpha1"
	kruisegameclientset "github.com/openkruise/kruise-game/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrlLog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// GAMESERVE_OWNER_LABEL ...
	GAMESERVE_OWNER_LABEL = "game.kruise.io/owner-gss"
)

type GamePatcher interface {
	// PatchGameServer patches the game server status
	PatchGameServer(ctx context.Context, gs *v1alpha1.GameServer) error

	ListGameServerByGSS(ctx context.Context, gssName, namespace string) (*v1alpha1.GameServerList, error)
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

func (o *okgGamePatcher) PatchGameServer(ctx context.Context, gs *v1alpha1.GameServer) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// 获取指定名称的gameserverset
		_, err := o.GameV1alpha1().GameServers(gs.Namespace).Update(ctx, gs, metav1.UpdateOptions{})
		if err != nil {
			ctrlLog.Log.Error(err, "Update gameserver err", "gsName", gs.Name, "gsNamespace", gs.Namespace)
			return err
		}
		return nil
	})
}

func (o *okgGamePatcher) ListGameServerByGSS(ctx context.Context, gssName, namespace string) (*v1alpha1.GameServerList, error) {
	gsList, err := o.GameV1alpha1().GameServers(namespace).List(ctx, metav1.ListOptions{
		ResourceVersion: "0",
		LabelSelector:   GAMESERVE_OWNER_LABEL + ":" + gssName,
	})
	if err != nil {
		ctrlLog.Log.Error(err, "List GameServerSet err")
		return nil, err
	}

	return gsList, nil
}
