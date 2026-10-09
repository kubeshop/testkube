package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/localinstall"
)

func TestResourceFix_PointsAtEachDockerAppsOwnSetting(t *testing.T) {
	tests := map[string]struct {
		docker localinstall.DockerResources
		cpu    bool
		want   string
	}{
		"Docker Desktop memory": {localinstall.DockerResources{Engine: "Docker Desktop", Memory: 4 << 30}, false, "Settings > Resources, set Memory to 6 GB"},
		"Colima CPU":            {localinstall.DockerResources{Engine: "colima", CPUs: 2}, true, "colima start --cpu 4"},
		"Linux memory":          {localinstall.DockerResources{Engine: "Ubuntu 24.04 LTS", Memory: 4 << 30}, false, "Free up memory"},
		"Docker unreadable":     {localinstall.DockerResources{}, false, "Give Docker 6 GB or more"},
		"enough on paper":       {localinstall.DockerResources{Engine: "Docker Desktop", Memory: 8 << 30}, false, "isn't enough free memory in the cluster"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Contains(t, resourceFix(tt.cpu, tt.docker), tt.want)
		})
	}
}
