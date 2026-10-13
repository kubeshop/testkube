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

type ExistingState int

const (
	TrialAbsent ExistingState = iota
	TrialSetUp
	// A node without answering services: a cancelled or failed install.
	TrialUnfinished
)

// Before step 1, so a rerun isn't promised a long wait.
func ExistingTrial(ctx context.Context) (state ExistingState, running bool) {
	cluster, err := NewCluster()
	if err != nil {
		return TrialAbsent, false
	}
	// A stuck Docker must not delay the plan.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	owned := func(ctx context.Context) (bool, Ports, error) {
		running, ports, _, err := cluster.owned(ctx)
		return running, ports, err
	}
	return existingTrial(ctx, owned, answers)
}

func existingTrial(ctx context.Context, owned func(context.Context) (bool, Ports, error),
	answers func(context.Context, string) bool) (ExistingState, bool) {
	running, ports, err := owned(ctx)
	switch {
	case err != nil:
		return TrialAbsent, false
	// A stopped node can't be probed; usually it was complete.
	case !running:
		return TrialSetUp, false
	}
	for _, name := range []string{"dashboard", "api", "login"} {
		if !answers(ctx, fmt.Sprintf("http://localhost:%d", ports[name])) {
			return TrialUnfinished, true
		}
	}
	return TrialSetUp, true
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
