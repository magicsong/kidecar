package sidecar

import (
	"context"

	"github.com/magicsong/okg-sidecar/pkg/assembler"
)

func main() {
	sidecar := assembler.NewSidecar(nil)
	ctx := context.TODO()
	if err := sidecar.Start(ctx); err != nil {
		panic(err)
	}
}
