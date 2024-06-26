package testworkflowcontroller

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
)

func SendControlCommand(ctx context.Context, podIP string, name string, body io.Reader) error {
	// TODO: add waiting for the started container + retries?
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("http://%s:%d/%s", podIP, constants.ControlServerPort, name), body)
	if err != nil {
		return err
	}
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return fmt.Errorf("control server error: %s", err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("control server error: status %d: %s", res.StatusCode, string(body))
	}
	return nil
}

func Pause(ctx context.Context, podIP string) error {
	return SendControlCommand(ctx, podIP, "pause", nil)
}

func Resume(ctx context.Context, podIP string) error {
	return SendControlCommand(ctx, podIP, "resume", nil)
}
