package routing

import (
	"context"
	"net/netip"
)

type Router interface {
	Ready(ctx context.Context) (bool, error)
	Resolve(ctx context.Context, key string, allowSelf bool, count int) (<-chan netip.AddrPort, error)
	Advertise(ctx context.Context, keys []string) error
}
