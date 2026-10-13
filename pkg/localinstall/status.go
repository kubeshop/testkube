package localinstall

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type TrialState int

const (
	TrialReady TrialState = iota
	TrialStarting
	TrialStopped
	TrialNotInstalled
	TrialNoDocker
	TrialElsewhere
	TrialNameTaken
)

type LocalStatus struct {
	State     TrialState
	Docker    Result
	Ports     Ports
	DataKept  bool
	Elsewhere ClusterElsewhereError
}

var errClusterNameTaken = errors.New("another cluster has our name")

// Before step 1, so a rerun isn't promised a long wait.
func ExistingTrial(ctx context.Context) (found, running bool) {
	cluster, err := NewCluster()
	if err != nil {
		return false, false
	}
	// A stuck Docker must not delay the plan.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	running, _, _, err = cluster.owned(ctx)
	return err == nil, running
}

// Reads only: a status check must never start anything.
func ReadStatus(ctx context.Context) LocalStatus {
	cluster, err := NewCluster()
	owned := func(ctx context.Context) (bool, Ports, error) {
		if err != nil {
			return false, nil, err
		}
		running, ports, _, err := cluster.owned(ctx)
		// A foreign node blocks install, so "not installed" would mislead.
		if errors.Is(err, ErrClusterNotOurs) {
			if _, inspectErr := cluster.run(ctx, "docker", "inspect", "-f", "{{.Name}}", nodeName); inspectErr == nil {
				return false, nil, errClusterNameTaken
			}
		}
		// Our mark without our node: maybe another Docker has it.
		if errors.Is(err, ErrClusterInspect) {
			if elsewhere := cluster.findElsewhere(ctx); elsewhere != nil {
				return false, nil, elsewhere
			}
		}
		return running, ports, err
	}
	return readStatus(ctx, NewChecker().CheckDocker, owned, answers, HasData)
}

func readStatus(ctx context.Context, docker func(context.Context) Result,
	owned func(context.Context) (bool, Ports, error), answers func(context.Context, string) bool, hasData func() bool) LocalStatus {
	s := LocalStatus{Docker: docker(ctx)}
	if s.Docker.Status == StatusFail {
		s.State = TrialNoDocker
		return s
	}
	running, ports, err := owned(ctx)
	switch {
	case errors.As(err, &s.Elsewhere):
		s.State = TrialElsewhere
	case errors.Is(err, errClusterNameTaken):
		s.State = TrialNameTaken
	// No mark or no node: nothing of ours exists.
	case err != nil:
		s.State, s.DataKept = TrialNotInstalled, hasData()
	case !running:
		s.State = TrialStopped
	// The API doesn't wait for login; check it too.
	case answers(ctx, fmt.Sprintf("http://localhost:%d", ports["dashboard"])) &&
		answers(ctx, fmt.Sprintf("http://localhost:%d", ports["api"])) &&
		answers(ctx, fmt.Sprintf("http://localhost:%d", ports["login"])):
		s.State, s.Ports = TrialReady, ports
	default:
		s.State, s.Ports = TrialStarting, ports
	}
	return s
}

// Any HTTP answer counts; the API refuses anonymous calls.
func answers(ctx context.Context, url string) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}
