package runners

import (
	"context"
	"fmt"
	"net"
)

type TCP struct {
	addr   string
	dialer net.Dialer
}

func NewTCP(addr string) *TCP {
	return &TCP{addr: addr}
}

func (t *TCP) Run(ctx context.Context) error {
	conn, err := t.dialer.DialContext(ctx, "tcp4", t.addr)
	if err != nil {
		return fmt.Errorf("dialing %q: %w", t.addr, err)
	}
	return conn.Close()
}
