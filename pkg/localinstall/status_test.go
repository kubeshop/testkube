package localinstall

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadStatus_NamesTheFirstThingInTheWay(t *testing.T) {
	ports := Ports{"dashboard": 8080, "api": 8090, "login": 5556}
	up := Result{Status: StatusPass}
	tests := map[string]struct {
		docker   Result
		running  bool
		ownedErr error
		answers  map[int]bool
		data     bool
		want     TrialState
		wantData bool
	}{
		"ready":                      {docker: up, running: true, answers: map[int]bool{8080: true, 8090: true, 5556: true}, want: TrialReady},
		"login not up yet":           {docker: up, running: true, answers: map[int]bool{8080: true, 8090: true}, want: TrialStarting},
		"dashboard up, API not yet":  {docker: up, running: true, answers: map[int]bool{8080: true}, want: TrialStarting},
		"stopped":                    {docker: up, want: TrialStopped},
		"uninstalled, data kept":     {docker: up, ownedErr: ErrClusterNotOurs, data: true, want: TrialNotInstalled, wantData: true},
		"in another Docker context":  {docker: up, ownedErr: ClusterElsewhereError{In: "colima", Active: "desktop-linux"}, data: true, want: TrialElsewhere},
		"mark but no cluster":        {docker: up, ownedErr: ErrClusterInspect, want: TrialNotInstalled},
		"someone else's testkube":    {docker: up, ownedErr: errClusterNameTaken, want: TrialNameTaken},
		"never installed":            {docker: up, ownedErr: ErrClusterNotOurs, want: TrialNotInstalled},
		"docker down hides the rest": {docker: Result{Status: StatusFail}, running: true, want: TrialNoDocker},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			owned := func(context.Context) (bool, Ports, error) { return tt.running, ports, tt.ownedErr }
			answers := func(_ context.Context, url string) bool {
				for port, ok := range tt.answers {
					if ok && url == fmt.Sprintf("http://localhost:%d", port) {
						return true
					}
				}
				return false
			}

			s := readStatus(context.Background(), func(context.Context) Result { return tt.docker }, owned, answers, func() bool { return tt.data })

			assert.Equal(t, tt.want, s.State)
			assert.Equal(t, tt.wantData, s.DataKept)
		})
	}
}

func TestAnswers_ARefusalStillMeansItsUp(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer s.Close()
	closed, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closedURL := "http://" + closed.Addr().String()
	closed.Close()

	assert.True(t, answers(context.Background(), s.URL))
	assert.False(t, answers(context.Background(), closedURL))
}

func TestExistingTrial_ACancelledInstallIsNotSetUp(t *testing.T) {
	ports := Ports{"dashboard": 8080, "api": 8090, "login": 5556}
	owned := func(running bool, err error) func(context.Context) (bool, Ports, error) {
		return func(context.Context) (bool, Ports, error) { return running, ports, err }
	}
	all := func(context.Context, string) bool { return true }
	none := func(context.Context, string) bool { return false }

	state, _ := existingTrial(context.Background(), owned(true, nil), all)
	assert.Equal(t, TrialSetUp, state)
	state, _ = existingTrial(context.Background(), owned(true, nil), none)
	assert.Equal(t, TrialUnfinished, state, "cluster up, Testkube never installed: 1.2 GB still ahead")
	state, _ = existingTrial(context.Background(), owned(false, nil), all)
	assert.Equal(t, TrialNodeStopped, state, "a stopped node proves nothing about Testkube")
	state, _ = existingTrial(context.Background(), owned(false, ErrClusterNotOurs), all)
	assert.Equal(t, TrialAbsent, state)
}
