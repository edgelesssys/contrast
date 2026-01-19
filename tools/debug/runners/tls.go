package runners

import (
	"context"
	"crypto/tls"
	"fmt"
)

type TLS struct {
	addr   string
	dialer *tls.Dialer
}

func NewTLS(addr string) *TLS {
	dialer := &tls.Dialer{Config: &tls.Config{InsecureSkipVerify: true}}
	return &TLS{addr: addr, dialer: dialer}
}

func (t *TLS) Run(ctx context.Context) error {
	conn, err := t.dialer.DialContext(ctx, "tcp4", t.addr)
	if err != nil {
		return fmt.Errorf("dialing %q: %w", t.addr, err)
	}
	return conn.Close()
}
