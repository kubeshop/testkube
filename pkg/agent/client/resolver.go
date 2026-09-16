package client

import (
	"context"
	"time"

	"google.golang.org/grpc/resolver"
)

// dnsRefreshBuilder wraps the stock "dns" resolver so it re-resolves on an interval.
//
// grpc-go's DNS resolver does not poll: after a successful lookup its watcher
// blocks until something calls ResolveNow, and only the load balancing policy does
// that, when a subchannel fails. A Control Plane replica added by a scale-up
// therefore stays invisible — it is never dialled and never receives a stream,
// which is how every agent ends up on the replicas that existed when it connected.
//
// Poking ResolveNow on a timer fixes that without breaking anything: re-resolution
// only grows the subchannel set, so established streams keep running on their
// current replica while new streams start reaching the new one. The stock resolver
// still rate-limits itself to dns.MinResolutionInterval (30s), so the interval here
// must be larger to take effect.
type dnsRefreshBuilder struct {
	inner    resolver.Builder
	interval time.Duration
}

func (b dnsRefreshBuilder) Scheme() string {
	return b.inner.Scheme()
}

func (b dnsRefreshBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	inner, err := b.inner.Build(target, cc, opts)
	if err != nil {
		return nil, err
	}
	if b.interval <= 0 {
		return inner, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &dnsRefreshResolver{Resolver: inner, cancel: cancel}
	go r.refresh(ctx, b.interval)
	return r, nil
}

type dnsRefreshResolver struct {
	resolver.Resolver
	cancel context.CancelFunc
}

func (r *dnsRefreshResolver) refresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.Resolver.ResolveNow(resolver.ResolveNowOptions{})
		}
	}
}

func (r *dnsRefreshResolver) Close() {
	r.cancel()
	r.Resolver.Close()
}

// refreshingDNSResolver returns the periodically re-resolving "dns" resolver to
// register on a ClientConn, or nil if the stock resolver is unavailable. Registering
// it per connection with grpc.WithResolvers keeps the global registry untouched.
func refreshingDNSResolver(interval time.Duration) resolver.Builder {
	inner := resolver.Get("dns")
	if inner == nil {
		return nil
	}
	return dnsRefreshBuilder{inner: inner, interval: interval}
}
