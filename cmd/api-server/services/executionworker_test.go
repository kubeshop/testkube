package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClusterDomainFromResolvConf(t *testing.T) {
	tests := []struct {
		name       string
		resolvConf string
		want       string
	}{
		{name: "reads the default domain", resolvConf: "search agent.svc.cluster.local svc.cluster.local cluster.local\noptions ndots:5\n", want: "cluster.local"},
		{name: "reads a custom domain", resolvConf: "nameserver 10.0.0.10\nsearch agent.svc.example.internal svc.example.internal\n", want: "example.internal"},
		{name: "ignores the search domain of another namespace", resolvConf: "search other.svc.example.internal\n", want: "cluster.local"},
		{name: "falls back without a search line", resolvConf: "nameserver 10.0.0.10\n", want: "cluster.local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, clusterDomainFromResolvConf(strings.NewReader(tt.resolvConf), "agent"))
		})
	}
}
