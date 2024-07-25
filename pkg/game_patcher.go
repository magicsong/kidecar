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
	// GAMESERVE_PROBE_LABEL ...
	GAMESERVE_PROBE_LABEL = "game.kruise.io/vke-probe-enabled"
	// GMAESERVER_PROBE_PORT_LABEL ...
	GMAESERVER_PROBE_PORT_LABEL = "game.kruise.io/vke-probe-port"
	// GAMESERVER_WANT_RESP ...
	GAMESERVER_WANT_RESP = "game.kruise.io/vke-want-resp"
	// GAMESERVER_NAMESPACE ...
	GAMESERVER_NAMESPACE = "default"
)

type GamePatcher interface {
	// PatchGameServer patches the game server status
	PatchGameServer(ctx context.Context, gs *v1alpha1.GameServer) error

	ListGameServersByProbeLabel(ctx context.Context) (*v1alpha1.GameServerList, error)

	GetGameServer(ctx context.Context, gsName, namespace string) (*v1alpha1.GameServer, error)
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

func (o *okgGamePatcher) GetGameServer(ctx context.Context, gsName, namespace string) (*v1alpha1.GameServer, error) {
	gs, err := o.GameV1alpha1().GameServers(namespace).Get(ctx, gsName, metav1.GetOptions{
		ResourceVersion: "0",
	})
	if err != nil {
		ctrlLog.Log.Error(err, "List GameServerSet err")
		return nil, err
	}

	return gs, nil
}

func (o *okgGamePatcher) ListGameServersByProbeLabel(ctx context.Context) (*v1alpha1.GameServerList, error) {
	gsList, err := o.GameV1alpha1().GameServers(GAMESERVER_NAMESPACE).List(ctx, metav1.ListOptions{
		ResourceVersion: "0",
		LabelSelector: metav1.FormatLabelSelector(
			&metav1.LabelSelector{
				MatchLabels: map[string]string{
					GAMESERVE_PROBE_LABEL: "true",
				},
			},
		),
	})
	if err != nil {
		ctrlLog.Log.Error(err, "List GameServerSet err")
		return nil, err
	}

	return gsList, nil
}
