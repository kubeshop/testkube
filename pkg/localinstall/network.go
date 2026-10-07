package localinstall

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Sites the tools, cluster and Testkube steps download from.
var networkProbes = []struct{ name, url string }{
	{"dl.k8s.io", "https://dl.k8s.io/"},
	{"get.helm.sh", "https://get.helm.sh/"},
	{"kind.sigs.k8s.io", "https://kind.sigs.k8s.io/"},
	{"Docker Hub", "https://registry-1.docker.io/v2/"},
	{"kubeshop.github.io", "https://kubeshop.github.io/helm-charts/index.yaml"},
	{"raw.githubusercontent.com", "https://raw.githubusercontent.com/"},
	{"github.com", "https://github.com/"},
}

const networkFix = "Testkube downloads from these during install. If you use a proxy,\n" +
	"set HTTPS_PROXY in this shell and in Docker's settings, then run again"

// Any HTTP answer counts: Docker Hub replies 401 to anonymous calls.
func (realHost) Reachable(ctx context.Context, url string) bool {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
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

// Warns only: Docker may use its own proxy or mirror.
func (c *Checker) CheckNetwork(ctx context.Context) Result {
	reachable := make([]bool, len(networkProbes))
	var wg sync.WaitGroup
	for i, p := range networkProbes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reachable[i] = c.host.Reachable(ctx, p.url)
		}()
	}
	wg.Wait()

	c.unreachable = nil
	for i, p := range networkProbes {
		if !reachable[i] {
			c.unreachable = append(c.unreachable, p.name)
		}
	}
	if len(c.unreachable) == 0 {
		return Result{Name: "network", Status: StatusPass, Detail: "download sites reachable"}
	}
	return Result{Name: "network", Status: StatusWarn, Detail: "can't reach " + strings.Join(c.unreachable, ", "), Fix: networkFix}
}
