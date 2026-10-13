package localinstall

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadStatus_NamesTheFirstThingInTheWay(t *testing.T) {
	ports := Ports{"dashboard": 8080, "api": 8090}
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
		"ready":                      {docker: up, running: true, answers: map[int]bool{8080: true, 8090: true}, want: TrialReady},
		"dashboard up, API not yet":  {docker: up, running: true, answers: map[int]bool{8080: true}, want: TrialStarting},
		"stopped":                    {docker: up, want: TrialStopped},
		"uninstalled, data kept":     {docker: up, ownedErr: ErrClusterNotOurs, data: true, want: TrialNotInstalled, wantData: true},
		"in another Docker context":  {docker: up, ownedErr: ClusterElsewhereError{In: "colima", Active: "desktop-linux"}, data: true, want: TrialElsewhere},
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
