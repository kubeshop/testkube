package controller

import (
	"context"

	constants2 "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/control"
)

func Pause(ctx context.Context, podIP string) error {
	client, err := control.NewClient(ctx, podIP, constants2.ControlServerPort)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Pause()
}

func Resume(ctx context.Context, podIP string) error {
	client, err := control.NewClient(ctx, podIP, constants2.ControlServerPort)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Resume()
}
