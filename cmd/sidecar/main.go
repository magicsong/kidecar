package main

import (
	"context"

	"github.com/magicsong/okg-sidecar/pkg/assembler"
	"github.com/magicsong/okg-sidecar/pkg/manager"
	"github.com/magicsong/okg-sidecar/pkg/plugins"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	logf.SetLogger(zap.New())
	log := logf.Log.WithName("manager-examples")
	sidecar := assembler.NewSidecar(nil)
	mgr, err := manager.NewManager()
	if err != nil {
		log.Error(err, "failed to create manager")
		panic(err)
	}
	sidecar.SetupWithManager(mgr)
	// add plugins
	for _, v := range plugins.PluginRegistry {
		if err := sidecar.AddPlugin(v); err != nil {
			log.Error(err, "failed to add plugin", "pluginName", v.Name())
			panic(err)
		}
	}
	ctx := context.TODO()
	if err := sidecar.Start(ctx); err != nil {
		panic(err)
	}
}
